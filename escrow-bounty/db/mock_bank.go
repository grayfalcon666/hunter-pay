package db

import (
	"context"
	"log"
	"time"
)

type MockBankClient struct{}

func (m *MockBankClient) Transfer(ctx context.Context, fromAccountID, toAccountID int64, amount int64, idempotencyKey string) error {
	log.Printf("[Mock Simplebank] 收到转账请求:")
	log.Printf("  -> 从账户: %v", fromAccountID)
	log.Printf("  -> 到账户: %v", toAccountID)
	log.Printf("  -> 金额: %d", amount)
	log.Printf("  -> 幂等键 (Idempotency Key): %s", idempotencyKey)
	time.Sleep(500 * time.Millisecond)
	log.Printf("[Mock Simplebank] 扣款成功！")
	return nil
}

func (m *MockBankClient) VerifyAccountOwner(ctx context.Context, accountID int64) error {
	return nil
}

func (m *MockBankClient) Freeze(ctx context.Context, employerAccountID, amount, bountyID int64, description, idempotencyKey string) error {
	log.Printf("[Mock Simplebank] 收到 Freeze 请求:")
	log.Printf("  -> 雇主账户: %v", employerAccountID)
	log.Printf("  -> 金额: %d", amount)
	log.Printf("  -> 悬赏ID: %d", bountyID)
	log.Printf("  -> 描述: %s", description)
	log.Printf("  -> 幂等键: %s", idempotencyKey)
	time.Sleep(500 * time.Millisecond)
	log.Printf("[Mock Simplebank] 冻结成功！")
	return nil
}

func (m *MockBankClient) Unfreeze(ctx context.Context, employerAccountID, amount, bountyID int64, description, idempotencyKey string) error {
	log.Printf("[Mock Simplebank] 收到 Unfreeze 请求:")
	log.Printf("  -> 雇主账户: %v", employerAccountID)
	log.Printf("  -> 金额: %d", amount)
	log.Printf("  -> 悬赏ID: %d", bountyID)
	log.Printf("  -> 描述: %s", description)
	log.Printf("  -> 幂等键: %s", idempotencyKey)
	time.Sleep(500 * time.Millisecond)
	log.Printf("[Mock Simplebank] 解冻成功！")
	return nil
}

func (m *MockBankClient) BountyPayout(ctx context.Context, employerAccountID, hunterAccountID, amount, bountyID int64, description, idempotencyKey string) error {
	log.Printf("[Mock Simplebank] 收到 BountyPayout 请求:")
	log.Printf("  -> 雇主账户: %v", employerAccountID)
	log.Printf("  -> 猎人账户: %v", hunterAccountID)
	log.Printf("  -> 金额: %d", amount)
	log.Printf("  -> 悬赏ID: %d", bountyID)
	log.Printf("  -> 描述: %s", description)
	log.Printf("  -> 幂等键: %s", idempotencyKey)
	time.Sleep(500 * time.Millisecond)
	log.Printf("[Mock Simplebank] 悬赏打款成功！")
	return nil
}
