package gapi

import (
	"context"

	db "simplebank/db/sqlc"
	"simplebank/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (server *Server) ListTransfers(ctx context.Context, req *pb.ListTransfersRequest) (*pb.ListTransfersResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	accountID := req.GetAccountId()
	page := req.GetPage()
	pageSize := req.GetPageSize()

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Verify account belongs to user
	account, err := server.store.GetAccount(ctx, accountID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "账户不存在: %v", err)
	}
	if account.Owner != authPayload.Username && authPayload.Username != "escrow_system" && authPayload.Username != "escrow" {
		return nil, status.Errorf(codes.PermissionDenied, "无权访问该账户的转账记录")
	}

	limit := int32(pageSize)
	offset := (int32(page) - 1) * int32(pageSize)

	// Get total count first
	total, err := server.store.CountAccountTransfers(ctx, accountID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询总数失败: %v", err)
	}

	// Get transfers with proper UNION ALL + global ordering
	rows, err := server.store.ListAccountTransfers(ctx, db.ListAccountTransfersParams{
		FromAccountID: accountID,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询转账记录失败: %v", err)
	}

	transfers := make([]*pb.Transfer, 0, len(rows))
	for _, t := range rows {
		tradeID := int64(0)
		if t.TradeID.Valid {
			tradeID = t.TradeID.Int64
		}
		desc := ""
		if t.Description.Valid {
			desc = t.Description.String
		}
		transfers = append(transfers, &pb.Transfer{
			Id:            t.ID,
			FromAccountId: t.FromAccountID,
			ToAccountId:   t.ToAccountID,
			Amount:        t.Amount,
			TradeType:     t.TradeType,
			TradeId:       tradeID,
			Description:   desc,
			CreatedAt:     timestamppb.New(t.CreatedAt),
		})
	}

	return &pb.ListTransfersResponse{
		Transfers: transfers,
		Total:     total,
		Page:      int32(page),
		PageSize:  int32(pageSize),
	}, nil
}
