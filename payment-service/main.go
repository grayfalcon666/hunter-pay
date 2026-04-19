package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/grayfalcon666/payment-service/api"
	"github.com/grayfalcon666/payment-service/db"
	"github.com/grayfalcon666/payment-service/gapi"
	"github.com/grayfalcon666/payment-service/mq"
	"github.com/grayfalcon666/payment-service/pb"
	"github.com/grayfalcon666/payment-service/token"
	"github.com/grayfalcon666/payment-service/util"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/smartwalle/alipay/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 1. 加载配置
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatalf("无法加载配置: %v", err)
	}

	// 2. 初始化数据库连接 (GORM)
	conn, err := gorm.Open(postgres.Open(config.DBSource), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法连接数据库: %v", err)
	}
	store := db.NewStore(conn)

	// 3. 初始化鉴权组件
	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		log.Fatalf("无法初始化 JWT Maker: %v", err)
	}

	// 4. 初始化 gRPC 客户端 (连接 SimpleBank 核心账本)
	systemToken, _ := tokenMaker.CreateToken(config.EscrowSystemUsername, 365*24*time.Hour)
	grpcConn, err := grpc.Dial(
		config.SimpleBankAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),                  // 阻塞直到连接建立
		grpc.WithTimeout(5*time.Second),   // 连接建立超时 5 秒
	)
	if err != nil {
		log.Fatalf("无法连接到 Simple Bank gRPC: %v", err)
	}
	defer grpcConn.Close()
	bankClient := db.NewGRPCBankClient(grpcConn, systemToken)

	// 5. 初始化支付宝 SDK 客户端
	alipayClient, err := alipay.New(config.AlipayAppID, config.AlipayPrivateKey, false)
	if err != nil {
		log.Fatalf("初始化支付宝 SDK 失败: %v", err)
	}
	_ = alipayClient.LoadAliPayPublicKey(config.AlipayPublicKey)

	// 6. 初始化 RabbitMQ 生产者
	producer, err := mq.NewProducer(config.RabbitMQURL)
	if err != nil {
		log.Fatalf("初始化 MQ 生产者失败: %v", err)
	}
	defer producer.Close()

	// 7. 启动提现后台 Worker (消费者)
	consumer, _ := mq.NewWithdrawalConsumer(config.RabbitMQURL, store, alipayClient, bankClient, config.PlatformEscrowAccountID)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := consumer.Start(ctx); err != nil {
			log.Fatalf("提现消费者启动失败: %v", err)
		}
	}()

	// 8. 启动服务器逻辑
	// 同时启动 gRPC 和 HTTP Gateway
	go runGatewayServer(config, store, alipayClient, producer, tokenMaker, bankClient)
	runGrpcServer(config, store, alipayClient, producer, tokenMaker, bankClient)
}

func runGrpcServer(config util.Config, store *db.Store, alipayClient *alipay.Client, producer *mq.Producer, tokenMaker *token.JWTMaker, bankClient *db.GRPCBankClient) {
	server, _ := gapi.NewServer(config, alipayClient, producer, store, tokenMaker, bankClient)

	grpcServer := grpc.NewServer()
	pb.RegisterPaymentServiceServer(grpcServer, server)
	reflection.Register(grpcServer) // 方便使用 gRPC UI 调试

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatal("cannot create listener", err)
	}

	log.Printf("gRPC Server 正在运行于 %s", listener.Addr().String())
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal("cannot start gRPC server", err)
	}
}

func runGatewayServer(config util.Config, store *db.Store, alipayClient *alipay.Client, producer *mq.Producer, tokenMaker *token.JWTMaker, bankClient *db.GRPCBankClient) {
	server, _ := gapi.NewServer(config, alipayClient, producer, store, tokenMaker, bankClient)

	// 1. 设置 grpc-gateway 的 JSON 序列化选项
	jsonOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames: true, // 使用 proto 里的字段名而不是驼峰
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true, // 忽略前端传来的未知字段
		},
	})

	grpcMux := runtime.NewServeMux(jsonOption)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. 将 gRPC Server 本地注册给 Gateway (无网络开销)
	if err := pb.RegisterPaymentServiceHandlerServer(ctx, grpcMux, server); err != nil {
		log.Fatal("cannot register handler server", err)
	}

	// 3. 混合路由：grpc-gateway 请求 + 原生 HTTP Webhook 请求
	mux := http.NewServeMux()

	// 挂载支付宝 Webhook
	webhook := api.NewWebhookServer(alipayClient, store, producer)
	mux.HandleFunc("/webhook/alipay", webhook.HandleAlipayWebhook)

	// 剩余所有请求交给 grpc-gateway 处理
	mux.Handle("/", grpcMux)

	listener, err := net.Listen("tcp", config.HTTPServerAddress)
	if err != nil {
		log.Fatal("cannot create HTTP listener", err)
	}

	log.Printf("HTTP Gateway Server 正在运行于 %s", listener.Addr().String())
	if err := http.Serve(listener, mux); err != nil {
		log.Fatal("cannot start HTTP gateway server", err)
	}
}
