package mq

const (
	WithdrawalQueue      = "withdrawal_process_queue"
	WithdrawalRoutingKey = "withdrawal.process"
)

type WithdrawalMessage struct {
	WithdrawalID   int64  `json:"withdrawal_id"`
	Username       string `json:"username"`
	AccountID      int64  `json:"account_id"`
	AlipayRealName string `json:"alipay_real_name"`
	Amount         int64  `json:"amount"`
	AlipayAccount  string `json:"alipay_account"`
	OutBizNo       string `json:"out_biz_no"`
}
