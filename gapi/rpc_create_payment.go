package gapi

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/grayfalcon666/payment-service/models"
	"github.com/grayfalcon666/payment-service/pb"
	"github.com/smartwalle/alipay/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) CreatePayment(ctx context.Context, req *pb.CreatePaymentRequest) (*pb.CreatePaymentResponse, error) {
	if req.GetAmount() <= 0 || req.GetAccountId() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "金额或账户 ID 不合法")
	}

	// 检查依赖服务是否可用（网关健康状态）
	if err := server.checkDependencies(ctx); err != nil {
		return nil, status.Errorf(codes.Unavailable, "系统暂时无法处理充值: %v", err)
	}

	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	outTradeNo := uuid.New().String()
	payment := &models.Payment{
		Username:   authPayload.Username,
		AccountID:  req.GetAccountId(),
		Amount:     req.GetAmount(),
		OutTradeNo: outTradeNo,
		Status:     "PENDING",
	}

	if err := server.store.CreatePayment(ctx, payment); err != nil {
		return nil, status.Errorf(codes.Internal, "创建订单失败: %v", err)
	}

	var p = alipay.TradePagePay{}
	p.NotifyURL = server.config.WebhookBaseURL + "/webhook/alipay"
	p.ReturnURL = server.config.FrontedBaseURL + "/payment/success"
	p.Subject = "Escrow 担保平台充值"
	p.OutTradeNo = outTradeNo
	p.TotalAmount = fmt.Sprintf("%.2f", float64(req.GetAmount())/100.0)
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"

	url, err := server.alipayClient.TradePagePay(p)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "生成支付链接失败: %v", err)
	}

	return &pb.CreatePaymentResponse{
		PayUrl: url.String(),
	}, nil
}
