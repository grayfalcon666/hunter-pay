package db

import (
	"context"

	"github.com/grayfalcon666/payment-service/models"
	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// 创建一笔支付订单
func (s *Store) CreatePayment(ctx context.Context, payment *models.Payment) error {
	return s.db.WithContext(ctx).Create(payment).Error
}

// 根据系统流水号查询订单
func (s *Store) GetPaymentByOutTradeNo(ctx context.Context, outTradeNo string) (*models.Payment, error) {
	var payment models.Payment
	err := s.db.WithContext(ctx).Where("out_trade_no = ?", outTradeNo).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// Webhook 成功后更新订单状态和支付宝流水号
func (s *Store) MarkPaymentSuccess(ctx context.Context, outTradeNo, alipayTradeNo string) error {
	return s.db.WithContext(ctx).
		Model(&models.Payment{}).
		Where("out_trade_no = ? AND status = 'PENDING'", outTradeNo). // 防止重复回调
		Updates(map[string]interface{}{
			"status":          "SUCCESS",
			"alipay_trade_no": alipayTradeNo,
		}).Error
}

// CreateWithdrawal 创建一条提现记录
func (s *Store) CreateWithdrawal(ctx context.Context, withdrawal *models.Withdrawal) error {
	return s.db.WithContext(ctx).Create(withdrawal).Error
}

// UpdateWithdrawalStatus 提现失败时，更新状态并记录错误原因
func (s *Store) UpdateWithdrawalStatus(ctx context.Context, outBizNo, status, errorMsg string) error {
	return s.db.WithContext(ctx).
		Model(&models.Withdrawal{}).
		Where("out_biz_no = ?", outBizNo).
		Updates(map[string]interface{}{
			"status":    status,
			"error_msg": errorMsg,
		}).Error
}

// TryUpdateWithdrawalStatusToProcessing 乐观锁：将 INIT → PROCESSING
// 如果当前状态不是 INIT（已被其他请求处理过），返回成功但 RowsAffected=0
// 用于幂等处理：重复的提现请求会被安全地忽略
func (s *Store) TryUpdateWithdrawalStatusToProcessing(ctx context.Context, outBizNo string) (rowsAffected int64, err error) {
	result := s.db.WithContext(ctx).
		Model(&models.Withdrawal{}).
		Where("out_biz_no = ? AND status = ?", outBizNo, "INIT").
		Update("status", "PROCESSING")
	return result.RowsAffected, result.Error
}

// TryClaimWithdrawal 消费者认领：尝试将 PROCESSING → REFUNDING
// 用于消费者端的幂等处理：如果已被其他消费者认领（状态已变），返回 RowsAffected=0
func (s *Store) TryClaimWithdrawal(ctx context.Context, outBizNo string) (rowsAffected int64, err error) {
	result := s.db.WithContext(ctx).
		Model(&models.Withdrawal{}).
		Where("out_biz_no = ? AND status = ?", outBizNo, "PROCESSING").
		Update("status", "REFUNDING")
	return result.RowsAffected, result.Error
}

// UpdateWithdrawalSuccess 提现成功时，更新状态并记录支付宝流水号
func (s *Store) UpdateWithdrawalSuccess(ctx context.Context, outBizNo, payFundOrderID string) error {
	return s.db.WithContext(ctx).
		Model(&models.Withdrawal{}).
		Where("out_biz_no = ?", outBizNo).
		Updates(map[string]interface{}{
			"status":            "SUCCESS",
			"pay_fund_order_id": payFundOrderID,
		}).Error
}
