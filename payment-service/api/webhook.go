package api

import (
	"log"
	"net/http"

	"github.com/grayfalcon666/payment-service/db"
	"github.com/grayfalcon666/payment-service/mq"
	"github.com/smartwalle/alipay/v3"
)

// WebhookServer 专门用于处理外部第三方的异步回调通知
type WebhookServer struct {
	alipayClient *alipay.Client
	store        *db.Store
	mqProducer   *mq.Producer
}

func NewWebhookServer(alipayClient *alipay.Client, store *db.Store, mqProducer *mq.Producer) *WebhookServer {
	return &WebhookServer{
		alipayClient: alipayClient,
		store:        store,
		mqProducer:   mqProducer,
	}
}

// HandleAlipayWebhook 处理支付宝支付成功的异步通知
func (s *WebhookServer) HandleAlipayWebhook(w http.ResponseWriter, r *http.Request) {
	// GET 请求仅用于 webhook 路径可达性健康检查（checkDependencies 发起）
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 1. 解析表单（POST/PUT 等）
	if err := r.ParseForm(); err != nil {
		log.Printf("解析表单失败: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 2. 支付宝签名验证 (RSA2)
	// 使用 r.Context() 以支持超时控制
	err := s.alipayClient.VerifySign(r.Context(), r.Form)
	if err != nil {
		log.Printf("支付宝 Webhook 验签失败: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	outTradeNo := r.Form.Get("out_trade_no")
	tradeStatus := r.Form.Get("trade_status")
	alipayTradeNo := r.Form.Get("trade_no")

	// 3. 业务状态判断
	if tradeStatus == "TRADE_SUCCESS" {
		// 查询本地订单，防止伪造单号请求
		payment, err := s.store.GetPaymentByOutTradeNo(r.Context(), outTradeNo)
		if err != nil {
			log.Printf("订单不存在: %s", outTradeNo)
			w.WriteHeader(http.StatusOK) // 返回 OK 停止支付宝重试
			return
		}

		// 幂等性校验：如果状态已是 SUCCESS，直接返回成功
		if payment.Status == "SUCCESS" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
			return
		}

		// 4. 更新本地数据库状态
		err = s.store.MarkPaymentSuccess(r.Context(), outTradeNo, alipayTradeNo)
		if err != nil {
			log.Printf("更新数据库订单状态失败: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// 5. 核心：通过 RabbitMQ 发送充值成功消息给 SimpleBank 账本
		msg := &mq.PaymentSuccessMessage{
			Username:   payment.Username,
			AccountID:  payment.AccountID,
			Amount:     payment.Amount,
			OutTradeNo: payment.OutTradeNo,
		}

		err = s.mqProducer.PublishPaymentSuccess(msg)
		if err != nil {
			log.Printf("发送 MQ 消息失败: %v", err)
			// 如果 MQ 失败，建议不写 success，让支付宝稍后重试回调
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Printf("充值闭环完成: 订单 %s, 用户 %s 已入库并发送 MQ", outTradeNo, payment.Username)
	}

	// 必须返回 success 字符串，否则支付宝会持续 24 小时不断回调
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
}
