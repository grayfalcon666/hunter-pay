package gapi

import (
	"context"
	"database/sql"

	db "simplebank/db/sqlc"
	"simplebank/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (server *Server) CreateLedgerEntry(ctx context.Context, req *pb.CreateLedgerEntryRequest) (*pb.CreateLedgerEntryResponse, error) {
	// No auth required for internal service calls (escrow-bounty calls this)
	// In production, use a service token or mTLS to authenticate

	var bountyID sql.NullInt64
	if req.BountyId != 0 {
		bountyID = sql.NullInt64{Int64: req.BountyId, Valid: true}
	}

	arg := db.CreateLedgerEntryParams{
		AccountID:   req.AccountId,
		BountyID:    bountyID,
		EventType:   req.EventType,
		Amount:      req.Amount,
		Counterparty: sql.NullString{String: req.Counterparty, Valid: req.Counterparty != ""},
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
	}

	entry, err := server.store.CreateLedgerEntry(ctx, arg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "创建账本记录失败: %v", err)
	}

	return &pb.CreateLedgerEntryResponse{
		Entry: &pb.LedgerEntry{
			Id:           entry.ID.String(),
			AccountId:    entry.AccountID,
			BountyId:     entry.BountyID.Int64,
			EventType:    entry.EventType,
			Amount:       entry.Amount,
			Counterparty: entry.Counterparty.String,
			Description:  entry.Description.String,
			CreatedAt:    timestamppb.New(entry.CreatedAt.Time),
		},
	}, nil
}

func (server *Server) ListAccountLedger(ctx context.Context, req *pb.ListAccountLedgerRequest) (*pb.ListAccountLedgerResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	accountID := req.AccountId
	page := req.Page
	pageSize := req.PageSize

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
		return nil, status.Errorf(codes.PermissionDenied, "无权访问该账户的账本")
	}

	limit := int32(pageSize)
	offset := (int32(page) - 1) * int32(pageSize)

	entries, err := server.store.ListLedgerEntriesByAccount(ctx, db.ListLedgerEntriesByAccountParams{
		AccountID: accountID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询账本失败: %v", err)
	}

	pbEntries := make([]*pb.LedgerEntry, len(entries))
	for i, e := range entries {
		pbEntries[i] = &pb.LedgerEntry{
			Id:           e.ID.String(),
			AccountId:    e.AccountID,
			BountyId:     e.BountyID.Int64,
			EventType:    e.EventType,
			Amount:       e.Amount,
			Counterparty: e.Counterparty.String,
			Description:  e.Description.String,
			CreatedAt:    timestamppb.New(e.CreatedAt.Time),
		}
	}

	return &pb.ListAccountLedgerResponse{
		LedgerEntries: pbEntries,
		Total:         int64(len(entries)),
		Page:          int32(page),
		PageSize:      int32(pageSize),
	}, nil
}

