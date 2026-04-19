package db

import (
	"context"
	"fmt"

	simplebankpb "github.com/grayfalcon666/escrow-bounty/simplebankpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type GRPCBankClient struct {
	client      simplebankpb.SimpleBankClient
	systemToken string
}

func NewGRPCBankClient(cc grpc.ClientConnInterface, systemToken string) *GRPCBankClient {
	return &GRPCBankClient{
		client:      simplebankpb.NewSimpleBankClient(cc),
		systemToken: systemToken,
	}
}

func (c *GRPCBankClient) Transfer(ctx context.Context, fromAccountID, toAccountID, amount int64, idempotencyKey string) error {
	var outgoingCtx context.Context

	md, ok := metadata.FromIncomingContext(ctx)

	if ok && len(md.Get("authorization")) > 0 {
		// 场景 A：前端用户主动触发 (如：Alice 发布悬赏)。将 Alice 的 Token 透传给下游 Simple Bank！
		outgoingCtx = metadata.NewOutgoingContext(context.Background(), md)
	} else {
		// 场景 B：微服务后台异步触发 (如：平台把钱结算给猎人)。此时没有前端 Token，使用系统 Token。
		systemMD := metadata.Pairs("authorization", "Bearer "+c.systemToken)
		outgoingCtx = metadata.NewOutgoingContext(context.Background(), systemMD)
	}

	req := &simplebankpb.TransferTxRequest{
		FromAccountId:  fromAccountID,
		ToAccountId:    toAccountID,
		Amount:         amount,
		IdempotencyKey: idempotencyKey,
	}

	_, err := c.client.Transfer(outgoingCtx, req)
	if err != nil {
		return fmt.Errorf("调用 Simple Bank 真实转账接口失败: %w", err)
	}

	return nil
}

// VerifyAccountOwner 用系统 Token 验证账户存在性（系统有权限访问所有账户）
func (c *GRPCBankClient) VerifyAccountOwner(ctx context.Context, accountID int64) error {
	systemMD := metadata.Pairs("authorization", "Bearer "+c.systemToken)
	outgoingCtx := metadata.NewOutgoingContext(context.Background(), systemMD)
	req := &simplebankpb.GetAccountRequest{
		Id: accountID,
	}

	// 呼叫 Simple Bank，系统 Token 有权限访问任意账户
	_, err := c.client.GetAccount(outgoingCtx, req)
	if err != nil {
		return fmt.Errorf("账户归属校验失败: %w", err)
	}

	return nil
}

// Freeze locks bounty amount in employer's account (BOUNTY_FREEZE)
func (c *GRPCBankClient) Freeze(ctx context.Context, employerAccountID, amount, bountyID int64, description, idempotencyKey string) error {
	systemMD := metadata.Pairs("authorization", "Bearer "+c.systemToken)
	outgoingCtx := metadata.NewOutgoingContext(context.Background(), systemMD)
	req := &simplebankpb.FreezeRequest{
		AccountId:       employerAccountID,
		Amount:          amount,
		BountyId:       bountyID,
		Description:    description,
		IdempotencyKey: idempotencyKey,
	}
	_, err := c.client.Freeze(outgoingCtx, req)
	if err != nil {
		return fmt.Errorf("调用 SimpleBank Freeze 失败: %w", err)
	}
	return nil
}

// Unfreeze unlocks bounty amount (BOUNTY_REFUND) - cancel bounty
func (c *GRPCBankClient) Unfreeze(ctx context.Context, employerAccountID, amount, bountyID int64, description, idempotencyKey string) error {
	systemMD := metadata.Pairs("authorization", "Bearer "+c.systemToken)
	outgoingCtx := metadata.NewOutgoingContext(context.Background(), systemMD)
	req := &simplebankpb.UnfreezeRequest{
		AccountId:       employerAccountID,
		Amount:          amount,
		BountyId:       bountyID,
		Description:    description,
		IdempotencyKey: idempotencyKey,
	}
	_, err := c.client.Unfreeze(outgoingCtx, req)
	if err != nil {
		return fmt.Errorf("调用 SimpleBank Unfreeze 失败: %w", err)
	}
	return nil
}

// BountyPayout pays hunter after bounty completion
func (c *GRPCBankClient) BountyPayout(ctx context.Context, employerAccountID, hunterAccountID, amount, bountyID int64, description, idempotencyKey string) error {
	systemMD := metadata.Pairs("authorization", "Bearer "+c.systemToken)
	outgoingCtx := metadata.NewOutgoingContext(context.Background(), systemMD)
	req := &simplebankpb.BountyPayoutRequest{
		EmployerAccountId: employerAccountID,
		HunterAccountId:   hunterAccountID,
		Amount:            amount,
		BountyId:         bountyID,
		Description:      description,
		IdempotencyKey:   idempotencyKey,
	}
	_, err := c.client.BountyPayout(outgoingCtx, req)
	if err != nil {
		return fmt.Errorf("调用 SimpleBank BountyPayout 失败: %w", err)
	}
	return nil
}

// ListAccounts 用用户的 Token 查询该用户所有账户
func (c *GRPCBankClient) ListAccounts(ctx context.Context) ([]*simplebankpb.Account, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md.Get("authorization")) == 0 {
		return nil, fmt.Errorf("未找到用户 Token")
	}
	outgoingCtx := metadata.NewOutgoingContext(context.Background(), md)

	// 分页取第一页，足够找到用户的账户
	req := &simplebankpb.ListAccountsRequest{
		PageId:   1,
		PageSize: 10,
	}

	resp, err := c.client.ListAccounts(outgoingCtx, req)
	if err != nil {
		return nil, fmt.Errorf("查询账户列表失败: %w", err)
	}

	return resp.Accounts, nil
}
