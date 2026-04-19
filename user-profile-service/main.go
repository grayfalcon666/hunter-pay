package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/grayfalcon666/user-profile-service/db"
	"github.com/grayfalcon666/user-profile-service/gapi"
	"github.com/grayfalcon666/user-profile-service/mq"
	"github.com/grayfalcon666/user-profile-service/pb"
	"github.com/grayfalcon666/user-profile-service/token"
	"github.com/grayfalcon666/user-profile-service/util"
	"github.com/grayfalcon666/user-profile-service/worker"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatalf("无法加载配置：%v", err)
	}

	db.InitDB(config.DBSource)
	store := db.NewStore(db.Client)

	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		log.Fatalf("无法创建 JWT Maker: %v", err)
	}

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 初始化 RabbitMQ 生产者（用于发布履约重算事件）
	amqpURL := config.RabbitMQURL
	if amqpURL == "" {
		amqpURL = "amqp://guest:guest@localhost:5672/"
	}
	var producer *mq.UserEventProducer
	producer, err = mq.NewUserEventProducer(amqpURL)
	if err != nil {
		log.Printf("RabbitMQ 生产者初始化失败：%v，将不启用事件发布功能", err)
	} else {
		defer producer.Close()
	}

	server := gapi.NewServer(config, store, tokenMaker, producer)

	// 启动 RabbitMQ 消费者（也使用同一个 producer）
	go runUserEventConsumer(ctx, config, store, producer)

	// 启动履约重算 MQ 消费者
	go runFulfillmentConsumer(ctx, config, store)

	// 启动 90 天活性衰减 Worker
	go runInactivityDecayWorker(ctx, config, store)

	// 启动 7 天评价结算 Worker
	go runReviewSettlerWorker(ctx, config, store)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		runGatewayServer(ctx, config, server)
	}()
	go func() {
		defer wg.Done()
		runGrpcServer(ctx, config, server)
	}()

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("收到退出信号，正在关闭...")
	cancel()

	wg.Wait()
	log.Println("服务已关闭")
}

// runUserEventConsumer 启动用户事件消费者
func runUserEventConsumer(ctx context.Context, config util.Config, store db.Store, producer *mq.UserEventProducer) {
	consumer, err := mq.NewUserEventConsumer(config.RabbitMQURL, store, producer)
	if err != nil {
		log.Printf("RabbitMQ 消费者初始化失败：%v", err)
		return
	}

	consumerCtx, consumerCancel := context.WithCancel(ctx)
	defer consumerCancel()

	go func() {
		<-ctx.Done()
		log.Println("收到退出信号，正在关闭 RabbitMQ 消费者...")
		consumerCancel()
	}()

	if err := consumer.Start(consumerCtx); err != nil {
		log.Printf("RabbitMQ 消费者运行错误：%v", err)
	}

	<-consumerCtx.Done()
	log.Println("RabbitMQ 消费者已停止")
}

func runFulfillmentConsumer(ctx context.Context, config util.Config, store db.Store) {
	amqpURL := config.RabbitMQURL
	if amqpURL == "" {
		amqpURL = "amqp://guest:guest@localhost:5672/"
	}

	consumer, err := mq.NewFulfillmentConsumer(amqpURL, store)
	if err != nil {
		log.Printf("FulfillmentConsumer 初始化失败：%v", err)
		return
	}
	defer consumer.Close()

	consumerCtx, consumerCancel := context.WithCancel(ctx)
	defer consumerCancel()

	go func() {
		<-ctx.Done()
		log.Println("收到退出信号，正在关闭 FulfillmentConsumer...")
		consumerCancel()
	}()

	if err := consumer.Start(consumerCtx); err != nil {
		log.Printf("FulfillmentConsumer 运行错误：%v", err)
	}

	<-consumerCtx.Done()
	log.Println("FulfillmentConsumer 已停止")
}

func runInactivityDecayWorker(ctx context.Context, config util.Config, store db.Store) {
	w := worker.NewInactivityDecayWorker(store, 24*time.Hour)
	w.Start(ctx)
}

func runReviewSettlerWorker(ctx context.Context, config util.Config, store db.Store) {
	w := worker.NewReviewSettlerWorker(store, 1*time.Hour)
	w.Start(ctx)
}

func runGrpcServer(ctx context.Context, config util.Config, server *gapi.Server) {
	grpcServer := grpc.NewServer()
	pb.RegisterProfileServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatalf("无法监听 gRPC 端口：%v", err)
	}

	log.Printf("启动 gRPC 服务，监听地址：%s", listener.Addr().String())

	// 在 goroutine 中启动服务
	errCh := make(chan error, 1)
	go func() {
		errCh <- grpcServer.Serve(listener)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			log.Printf("gRPC 服务运行失败：%v", err)
		}
	case <-ctx.Done():
		grpcServer.GracefulStop()
		log.Println("gRPC 服务已关闭")
	}
}

func runGatewayServer(ctx context.Context, config util.Config, server *gapi.Server) {
	grpcMux := runtime.NewServeMux()

	err := pb.RegisterProfileServiceHandlerServer(ctx, grpcMux, server)
	if err != nil {
		log.Fatalf("无法注册 Gateway 处理器：%v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", grpcMux)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Accept"},
		ExposedHeaders:   []string{"Grpc-Metadata-Authorization"},
		AllowCredentials: true,
	})

	handler := corsHandler.Handler(mux)

	listener, err := net.Listen("tcp", config.HTTPServerAddress)
	if err != nil {
		log.Fatalf("无法监听 HTTP 端口：%v", err)
	}

	log.Printf("启动 HTTP Gateway 服务，监听地址：%s", listener.Addr().String())

	errCh := make(chan error, 1)
	go func() {
		errCh <- http.Serve(listener, handler)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			log.Printf("HTTP Gateway 服务运行失败：%v", err)
		}
	case <-ctx.Done():
		log.Println("HTTP Gateway 服务已关闭")
	}
}
