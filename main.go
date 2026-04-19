package main

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"simplebank/api"
	db "simplebank/db/sqlc"
	"simplebank/doc"
	"simplebank/gapi"
	"simplebank/mail"
	"simplebank/mq"
	"simplebank/pb"
	"simplebank/util"
	"simplebank/worker"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hibiken/asynq"
	_ "github.com/lib/pq"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	var handler slog.Handler
	if config.Environment == "production" {
		handler = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}
	logger := slog.New(handler)
	slog.SetDefault(slog.New(handler))

	// 把标准库的 log 统一重定向到 slog
	// 这样 redis 包里调用的 log.Printf 就会自动变成 slog 的 JSON 输出
	l := slog.NewLogLogger(logger.Handler(), slog.LevelError)
	log.SetOutput(l.Writer())
	log.SetFlags(0)

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()

	store := db.NewStore(conn)

	consumer, err := mq.NewConsumer(config.RabbitMQURL, store)
	if err != nil {
		log.Fatalf("无法初始化 RabbitMQ 消费者：%v", err)
	}

	// 初始化用户事件生产者
	userProducer, err := mq.NewUserProducer(config.RabbitMQURL)
	if err != nil {
		log.Printf("RabbitMQ 用户事件生产者初始化失败：%v", err)
	}

	mailer := mail.NewGmailSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)

	redisOpt := asynq.RedisClientOpt{
		Addr: config.RedisAddress,
	}
	taskDistributor := worker.NewRedisTaskDistributor(redisOpt)

	waitGroup, ctx := errgroup.WithContext(ctx)

	err = consumer.Start(ctx)
	if err != nil {
		log.Fatalf("无法启动消费者：%v", err)
	}

	// 启动用户账户消费者（Saga 协调器）
	// 创建适配器以连接 db.Store 和 UserAccountStoreInterface
	userAccountStoreAdapter := &userAccountStoreAdapter{store: store}
	userAccountConsumer, err := mq.NewUserAccountConsumer(config.RabbitMQURL, userAccountStoreAdapter)
	if err != nil {
		log.Printf("RabbitMQ 用户账户消费者初始化失败：%v", err)
	} else {
		go func() {
			if err := userAccountConsumer.Start(ctx); err != nil {
				log.Printf("RabbitMQ 用户账户消费者启动失败：%v", err)
			}
		}()
	}

	go runGrpcServer(ctx, waitGroup, config, store, taskDistributor)
	go runTaskProcessor(ctx, waitGroup, redisOpt, store, mailer)
	runGatewayServer(ctx, waitGroup, config, store, taskDistributor, userProducer)
	// runGinServer(ctx, waitGroup, config, store)

	if err := waitGroup.Wait(); err != nil {
		log.Fatal("service exit with error:", err)
	}
	log.Println("all services stopped gracefully")
}

func runGatewayServer(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	store db.Store,
	distributor worker.TaskDistributor,
	userProducer *mq.UserProducer,
) {
	server, err := gapi.NewServer(config, store, distributor)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	// 设置用户事件生产者
	if userProducer != nil {
		server.SetUserProducer(userProducer)
	}

	// 设置 JSON 解析选项
	jsonOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames: true, // 使用 proto 里的字段名
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true, // 忽略前端传来的多余字段
		},
	})

	grpcMux := runtime.NewServeMux(jsonOption)

	err = pb.RegisterSimpleBankHandlerServer(ctx, grpcMux, server)
	if err != nil {
		log.Fatal("cannot register handler server:", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", grpcMux)

	subFS, err := fs.Sub(doc.SwaggerFiles, "swagger")
	if err != nil {
		log.Fatal("cannot create sub filesystem:", err)
	}

	fsServer := http.FileServer(http.FS(subFS))

	mux.Handle("/swagger/", http.StripPrefix("/swagger/", fsServer))

	listener, err := net.Listen("tcp", config.HTTPServerAddress)
	if err != nil {
		log.Fatal("cannot create listener:", err)
	}

	handler := gapi.HttpLogger(mux)

	httpServer := &http.Server{
		Handler: handler,
	}

	waitGroup.Go(func() error {
		log.Printf("start HTTP gateway server at %s", listener.Addr().String())
		if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP gateway server failed to serve: %v", err)
			return err
		}
		return nil
	})

	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Println("graceful shutdown HTTP gateway server")

		// 优雅关闭：处理完当前请求后关闭，超时 10 秒
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("failed to shutdown HTTP gateway server: %v", err)
			return err
		}

		log.Println("HTTP gateway server stopped")
		return nil
	})
}

func runGinServer(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	store db.Store,
) {
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	httpServer := &http.Server{
		Addr:    config.HTTPServerAddress,
		Handler: server.GetRouter(),
	}

	waitGroup.Go(func() error {
		log.Printf("start Gin server at %s", config.HTTPServerAddress)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Gin server failed to serve: %v", err)
			return err
		}
		return nil
	})

	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Println("graceful shutdown Gin server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("failed to shutdown Gin server: %v", err)
			return err
		}

		log.Println("Gin server stopped")
		return nil
	})
}

func runGrpcServer(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	store db.Store,
	distributor worker.TaskDistributor,
) {
	server, err := gapi.NewServer(config, store, distributor)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	grpcLogger := grpc.UnaryInterceptor(gapi.GrpcLogger)
	grpcServer := grpc.NewServer(grpcLogger)

	pb.RegisterSimpleBankServer(grpcServer, server)

	// 注册反射 (Reflection)
	// 它允许客户端（如 Evans 或 Postman）动态获取 API 定义
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatal("cannot create listener:", err)
	}

	waitGroup.Go(func() error {
		log.Printf("start gRPC server at %s", listener.Addr().String())
		if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Fatal("cannot start gRPC server:", err)
			return err
		}
		return nil
	})

	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Println("graceful shutdown gRPC server")
		grpcServer.GracefulStop()
		log.Println("gRPC server stopped")
		return nil
	})
}

func runTaskProcessor(
	ctx context.Context,
	waitGroup *errgroup.Group,
	redisOpt asynq.RedisClientOpt,
	store db.Store,
	mailer mail.EmailSender,
) {
	taskProcessor := worker.NewRedisTaskProcessor(redisOpt, store, mailer)

	waitGroup.Go(func() error {
		log.Printf("start task processor")
		if err := taskProcessor.Start(); err != nil {
			log.Fatalf("failed to start task processor: %v", err)
			return err
		}
		return nil
	})

	waitGroup.Go(func() error {
		//给协程阻塞信号 除非关闭了 ctx.Done() 通道 才会执行下面语句
		<-ctx.Done()

		taskProcessor.Shutdown()
		log.Println("task processor stopped")
		return nil
	})
}

// userAccountStoreAdapter 将 db.Store 适配为 mq.UserAccountStoreInterface
type userAccountStoreAdapter struct {
	store db.Store
}

func (a *userAccountStoreAdapter) CreateUserAccount(ctx context.Context, arg mq.CreateUserAccountParams) (int64, error) {
	return a.store.CreateUserAccount(ctx, db.CreateUserAccountParams{
		Owner:    arg.Owner,
		Currency: arg.Currency,
	})
}

func (a *userAccountStoreAdapter) UpdateUserStatus(ctx context.Context, arg mq.UpdateUserStatusParams) error {
	failedReason := ""
	if arg.FailedReason.Valid {
		failedReason = arg.FailedReason.String
	}
	return a.store.UpdateUserStatus(ctx, db.UpdateUserStatusParams{
		Status:       arg.Status.String,
		RequestID:    arg.RequestID.String,
		FailedReason: failedReason,
		Username:     arg.Username,
	})
}

func (a *userAccountStoreAdapter) CheckEventProcessed(ctx context.Context, requestID string) (bool, error) {
	return a.store.CheckEventProcessed(ctx, requestID)
}

func (a *userAccountStoreAdapter) MarkEventProcessed(ctx context.Context, arg mq.MarkEventProcessedParams) error {
	return a.store.MarkEventProcessed(ctx, db.MarkEventProcessedParams{
		RequestID: arg.RequestID,
		Username:  arg.Username,
		EventType: arg.EventType,
		Payload:   arg.Payload,
	})
}
