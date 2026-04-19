package gapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grayfalcon666/payment-service/db"
	"github.com/grayfalcon666/payment-service/mq"
	"github.com/grayfalcon666/payment-service/pb"
	"github.com/grayfalcon666/payment-service/token"
	"github.com/grayfalcon666/payment-service/util"
	"github.com/smartwalle/alipay/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedPaymentServiceServer
	config       util.Config
	bankClient   *db.GRPCBankClient
	alipayClient *alipay.Client
	mqProducer   *mq.Producer
	store        *db.Store
	tokenMaker   *token.JWTMaker
}

func NewServer(config util.Config, alipayClient *alipay.Client, producer *mq.Producer, store *db.Store, tokenMaker *token.JWTMaker, bankClient *db.GRPCBankClient) (*Server, error) {
	server := &Server{
		config:       config,
		alipayClient: alipayClient,
		mqProducer:   producer,
		store:        store,
		tokenMaker:   tokenMaker,
		bankClient:   bankClient,
	}
	return server, nil
}

func (server *Server) verifyUserAndAccount(ctx context.Context, accountID int64) (*token.Payload, error) {
	payload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	err = server.bankClient.VerifyAccountOwner(ctx, accountID)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "账户校验失败: %v", err)
	}

	return payload, nil
}

// checkDependencies 检查依赖服务是否可用（网关健康状态 + 支付宝回调连通性）
func (server *Server) checkDependencies(ctx context.Context) error {
	// 1. 检查 Gateway 健康状态（间接验证 webhook 回调路径是否通畅）
	gatewayURL := server.config.GatewayURL + "/health"
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", gatewayURL, nil)
	if err != nil {
		return fmt.Errorf("创建网关检查请求失败: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("网关服务不可用，无法处理充值: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("网关服务状态异常: %d", resp.StatusCode)
	}

	// 2. 验证支付宝能否访问到我们的 webhook URL
	webhookURL := server.config.WebhookBaseURL + "/webhook/alipay"
	webhookReq, err := http.NewRequestWithContext(ctx, "GET", webhookURL, nil)
	if err != nil {
		return fmt.Errorf("创建 webhook 检查请求失败: %w", err)
	}

	webhookResp, err := client.Do(webhookReq)
	if err != nil {
		return fmt.Errorf("支付回调地址不可达，支付宝无法通知: %v", err)
	}
	webhookResp.Body.Close()

	// webhook URL 必须外网可达（任何隧道方案：Cloudflare Tunnel、frp、花生棒等）
	if webhookResp.StatusCode >= 400 {
		return fmt.Errorf("支付回调地址不可达 (HTTP %d)，请检查网络和隧道状态", webhookResp.StatusCode)
	}

	return nil
}
