package db

import (
	"context"
	"fmt"

	simplebankpb "github.com/grayfalcon666/payment-service/simplebankpb"
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

func (c *GRPCBankClient) Transfer(ctx context.Context, fromAccountID, toAccountID, amount int64, idempotencyKey string, tradeType ...string) error {
	var outgoingCtx context.Context

	md, ok := metadata.FromIncomingContext(ctx)

	if ok && len(md.Get("authorization")) > 0 {
		outgoingCtx = metadata.NewOutgoingContext(context.Background(), md)
	} else {
		systemMD := metadata.Pairs("authorization", "Bearer "+c.systemToken)
		outgoingCtx = metadata.NewOutgoingContext(context.Background(), systemMD)
	}

	req := &simplebankpb.TransferTxRequest{
		FromAccountId:  fromAccountID,
		ToAccountId:    toAccountID,
		Amount:         amount,
		IdempotencyKey: idempotencyKey,
	}
	if len(tradeType) > 0 {
		req.TradeType = tradeType[0]
	}

	_, err := c.client.Transfer(outgoingCtx, req)
	if err != nil {
		return fmt.Errorf("调用 Simple Bank 转账接口失败: %w", err)
	}

	return nil
}

// 注意这里的参数，只有 ctx 和 accountID
func (c *GRPCBankClient) VerifyAccountOwner(ctx context.Context, accountID int64) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md.Get("authorization")) == 0 {
		return fmt.Errorf("未找到鉴权 Token，无法校验账户")
	}

	outgoingCtx := metadata.NewOutgoingContext(context.Background(), md)
	req := &simplebankpb.GetAccountRequest{
		Id: accountID,
	}

	_, err := c.client.GetAccount(outgoingCtx, req)
	if err != nil {
		return fmt.Errorf("账户归属校验失败: %w", err)
	}

	return nil
}

func (c *GRPCBankClient) Freeze(ctx context.Context, accountID, amount int64, idempotencyKey string) error {
	outgoingCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+c.systemToken))
	_, err := c.client.Freeze(outgoingCtx, &simplebankpb.FreezeRequest{
		AccountId:     accountID,
		Amount:        amount,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return fmt.Errorf("调用 Simple Bank Freeze 接口失败: %w", err)
	}
	return nil
}

func (c *GRPCBankClient) Unfreeze(ctx context.Context, accountID, amount int64, idempotencyKey string) error {
	outgoingCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+c.systemToken))
	_, err := c.client.Unfreeze(outgoingCtx, &simplebankpb.UnfreezeRequest{
		AccountId:     accountID,
		Amount:        amount,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return fmt.Errorf("调用 Simple Bank Unfreeze 接口失败: %w", err)
	}
	return nil
}

func (c *GRPCBankClient) WithdrawFromFrozen(ctx context.Context, accountID, amount int64, idempotencyKey string, description string) error {
	outgoingCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+c.systemToken))
	_, err := c.client.WithdrawFromFrozen(outgoingCtx, &simplebankpb.WithdrawFromFrozenRequest{
		AccountId:     accountID,
		Amount:        amount,
		IdempotencyKey: idempotencyKey,
		Description:   description,
	})
	if err != nil {
		return fmt.Errorf("调用 Simple Bank WithdrawFromFrozen 接口失败: %w", err)
	}
	return nil
}
