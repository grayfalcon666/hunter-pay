package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/grayfalcon666/escrow-bounty/db"
	"github.com/grayfalcon666/escrow-bounty/gapi"
	"github.com/grayfalcon666/escrow-bounty/mq"
	"github.com/grayfalcon666/escrow-bounty/pb"
	"github.com/grayfalcon666/escrow-bounty/token"
	"github.com/grayfalcon666/escrow-bounty/util"
	"github.com/grayfalcon666/escrow-bounty/wshub"
	"github.com/grayfalcon666/escrow-bounty/worker"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatalf("无法加载配置: %v", err)
	}

	db.InitDB(config.DBSource)
	store := db.NewStore(db.Client)

	// 初始化 RabbitMQ Producer
	producer, err := mq.NewProfileUpdateProducer(config.RabbitMQURL)
	if err != nil {
		log.Fatalf("无法初始化 RabbitMQ Producer: %v", err)
	}
	defer producer.Close()

	// 初始化履约重算 Producer
	fulfillmentProducer, err := mq.NewFulfillmentRecalcProducer(config.RabbitMQURL)
	if err != nil {
		log.Fatalf("无法初始化 FulfillmentRecalcProducer: %v", err)
	}
	defer fulfillmentProducer.Close()

	// 启动 Outbox Worker
	outboxWorker := mq.NewOutboxWorker(store, producer, 2*time.Second, 100)
	go outboxWorker.Start(context.Background())

	// 启动履约重算 Outbox Worker
	fulfillmentWorker := worker.NewFulfillmentOutboxWorker(store, fulfillmentProducer, 2*time.Second, 100)
	go fulfillmentWorker.Start(context.Background())

	// 启动过期检测 Worker
	deadlineWorker := worker.NewDeadlineChecker(store, 1*time.Minute, 100)
	go deadlineWorker.Start(context.Background())

	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		log.Fatalf("无法创建 JWT Maker: %v", err)
	}

	conn, err := grpc.NewClient(config.SimpleBankAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("无法连接到 Simple Bank: %v", err)
	}
	defer conn.Close()

	// employerToken, _ := tokenMaker.CreateToken("101", 24*time.Hour) // 模拟雇主 ID 为 101
	// hunterToken, _ := tokenMaker.CreateToken("102", 24*time.Hour)   // 模拟猎人 ID 为 102

	// log.Println("========================================")
	// log.Println("本地测试用 Tokens (有效期 24 小时):")
	// log.Printf("雇主 Token (用于发布悬赏, ID: 101):\nBearer %s\n\n", employerToken)
	// log.Printf("猎人 Token (用于接单申请, ID: 102):\nBearer %s\n", hunterToken)
	// log.Println("========================================")

	systemToken, _ := tokenMaker.CreateToken(config.EscrowSystemUsername, 365*24*time.Hour)

	bankClient := db.NewGRPCBankClient(conn, systemToken)

	profileConn, err := grpc.NewClient(config.UserProfileServiceAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("无法连接到 User Profile Service: %v", err)
	}
	defer profileConn.Close()

	profileClient := db.NewRawProfileClient(profileConn, systemToken)

	// Start WebSocket hub (needs store before server)
	wsHub := wshub.NewHub(store)
	go wsHub.Run()

	wsHandler := wshub.NewHandler(wsHub, tokenMaker)

	server := gapi.NewServer(config, store, bankClient, tokenMaker, profileClient, wsHub)
	go runWebSocketServer(config, wsHandler)

	go runGatewayServer(config, server)
	runGrpcServer(config, server)
}

func runGrpcServer(config util.Config, server pb.EscrowBountyServiceServer) {
	grpcServer := grpc.NewServer()
	pb.RegisterEscrowBountyServiceServer(grpcServer, server)
	reflection.Register(grpcServer) // 开启反射

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatalf("无法监听 gRPC 端口: %v", err)
	}

	log.Printf("启动 gRPC 服务，监听地址: %s", listener.Addr().String())
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("gRPC 服务运行失败: %v", err)
	}
}

func runGatewayServer(config util.Config, server *gapi.Server) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpcMux := runtime.NewServeMux()

	err := pb.RegisterEscrowBountyServiceHandlerServer(ctx, grpcMux, server)
	if err != nil {
		log.Fatalf("无法注册 Gateway 处理器: %v", err)
	}

	// 初始化图片上传配置
	gapi.InitUpload(config)

	// 混合ServeMux：
	// - POST/DELETE /upload, /upload/* -> 上传/删除处理器
	// - GET /uploads/* -> 静态文件服务（图片预览）
	// - /v1/* -> grpc-gateway
	mux := http.NewServeMux()

	// 上传路由：同时注册 /upload 和 /upload/（Go ServeMux 会自动处理尾部重定向）
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			server.UploadImageHandle(w, r)
		} else if r.Method == http.MethodDelete {
			server.DeleteImageHandle(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/upload/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			server.UploadImageHandle(w, r)
		} else if r.Method == http.MethodDelete {
			server.DeleteImageHandle(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	// 静态文件服务：/uploads/YYYY/MM/filename.jpg
	staticHandler := http.StripPrefix("/uploads/", http.FileServer(http.Dir(gapi.GetUploadBasePath())))
	mux.Handle("/uploads/", staticHandler)

	// gRPC Gateway
	mux.Handle("/v1/", grpcMux)

	// 配置 CORS
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
		log.Fatalf("无法监听 HTTP 端口: %v", err)
	}

	log.Printf("启动 HTTP Gateway 服务，监听地址: %s", listener.Addr().String())
	err = http.Serve(listener, handler)
	if err != nil {
		log.Fatalf("HTTP Gateway 服务运行失败: %v", err)
	}
}

func runWebSocketServer(config util.Config, handler http.Handler) {
	listener, err := net.Listen("tcp", config.WSServerAddress)
	if err != nil {
		log.Fatalf("无法监听 WebSocket 端口: %v", err)
	}
	log.Printf("启动 WebSocket 服务，监听地址: %s", listener.Addr().String())
	err = http.Serve(listener, handler)
	if err != nil {
		log.Fatalf("WebSocket 服务运行失败: %v", err)
	}
}
