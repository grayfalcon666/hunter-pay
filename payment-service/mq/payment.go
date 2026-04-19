package mq

const (
	PaymentQueue      = "payment_success_queue"
	PaymentRoutingKey = "payment.success"
)

type PaymentSuccessMessage struct {
	Username   string `json:"username"`
	AccountID  int64  `json:"account_id"`
	Amount     int64  `json:"amount"`
	OutTradeNo string `json:"out_trade_no"`
}
