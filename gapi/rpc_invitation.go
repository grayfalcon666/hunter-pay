package gapi

import (
	"context"
	"strings"

	"github.com/grayfalcon666/escrow-bounty/models"
	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateInvitation 雇主邀请猎人接单
func (server *Server) CreateInvitation(ctx context.Context, req *pb.CreateInvitationRequest) (*pb.CreateInvitationResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetBountyId() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "非法的悬赏 ID")
	}
	if req.GetHunterUsername() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "猎人用户名不能为空")
	}

	// 验证悬赏是否存在且属于当前用户
	bounty, err := server.store.GetBountyByID(ctx, req.GetBountyId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "悬赏不存在: %v", err)
	}
	if bounty.EmployerUsername != authPayload.Username {
		return nil, status.Errorf(codes.PermissionDenied, "只能给自己的悬赏发送邀请")
	}
	if bounty.Status != models.BountyStatusPending {
		return nil, status.Errorf(codes.FailedPrecondition, "当前悬赏状态为 %s，无法邀请猎人", bounty.Status)
	}

	// 检查是否已有该猎人的申请
	existingApps, err := server.listApplications(ctx, req.GetBountyId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询申请失败: %v", err)
	}
	for _, app := range existingApps {
		if app.HunterUsername == req.GetHunterUsername() {
			return nil, status.Errorf(codes.AlreadyExists, "该猎人已经申请过此悬赏")
		}
	}

	inv := &models.Invitation{
		BountyID:       req.GetBountyId(),
		PosterUsername: authPayload.Username,
		HunterUsername: req.GetHunterUsername(),
		Status:         models.InvitationStatusPending,
	}

	created, err := server.store.CreateInvitation(ctx, inv)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "SQLSTATE 23505") {
			return nil, status.Errorf(codes.AlreadyExists, "已经邀请过该猎人接单")
		}
		return nil, status.Errorf(codes.Internal, "创建邀请失败: %v", err)
	}

	return &pb.CreateInvitationResponse{
		Invitation: convertInvitation(created),
	}, nil
}

// GetMyInvitations 猎人查看收到的邀请
func (server *Server) GetMyInvitations(ctx context.Context, req *pb.GetMyInvitationsRequest) (*pb.GetMyInvitationsResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	var filterStatus models.InvitationStatus
	if req.GetStatus() != "" {
		filterStatus = models.InvitationStatus(req.GetStatus())
	}

	limit := int(req.GetPageSize())
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if req.GetPageId() > 1 {
		offset = int(req.GetPageId()-1) * limit
	}

	invs, err := server.store.ListInvitationsByHunter(ctx, authPayload.Username, filterStatus, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询邀请失败: %v", err)
	}

	return &pb.GetMyInvitationsResponse{
		Invitations: convertInvitations(invs),
	}, nil
}

// RespondToInvitation 猎人响应邀请（接受/拒绝）
func (server *Server) RespondToInvitation(ctx context.Context, req *pb.RespondToInvitationRequest) (*pb.RespondToInvitationResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	inv, err := server.store.GetInvitationByID(ctx, req.GetInvitationId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "邀请不存在: %v", err)
	}
	if inv.HunterUsername != authPayload.Username {
		return nil, status.Errorf(codes.PermissionDenied, "只能响应发给自己的邀请")
	}
	if inv.Status != models.InvitationStatusPending {
		return nil, status.Errorf(codes.FailedPrecondition, "邀请已处理，当前状态: %s", inv.Status)
	}

	if req.GetAccept() {
		// 猎人接受邀请：创建申请记录
		accounts, err := server.bankClient.ListAccounts(ctx)
		if err != nil || len(accounts) == 0 {
			return nil, status.Errorf(codes.FailedPrecondition, "猎人账户不存在")
		}
		hunterAccountID := accounts[0].GetId()

		app := &models.BountyApplication{
			BountyID:        inv.BountyID,
			HunterUsername:  authPayload.Username,
			HunterAccountID: hunterAccountID,
			Status:          models.AppStatusApplied,
		}
		if err := server.store.CreateApplication(ctx, app); err != nil {
			if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "SQLSTATE 23505") {
				return nil, status.Errorf(codes.AlreadyExists, "你已经申请过该悬赏")
			}
			return nil, status.Errorf(codes.Internal, "创建申请失败: %v", err)
		}

		// 更新邀请状态为已接受
		updated, err := server.store.UpdateInvitationStatus(ctx, inv.ID, models.InvitationStatusAccepted)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "更新邀请状态失败: %v", err)
		}

		// 获取悬赏详情
		bounty, _ := server.store.GetBountyByID(ctx, inv.BountyID)
		return &pb.RespondToInvitationResponse{
			Invitation: convertInvitation(updated),
			Bounty:    convertBounty(bounty),
		}, nil
	} else {
		// 猎人拒绝邀请
		updated, err := server.store.UpdateInvitationStatus(ctx, inv.ID, models.InvitationStatusDeclined)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "更新邀请状态失败: %v", err)
		}
		return &pb.RespondToInvitationResponse{
			Invitation: convertInvitation(updated),
		}, nil
	}
}

// GetMySentInvitations 雇主查看发出的邀请
func (server *Server) GetMySentInvitations(ctx context.Context, req *pb.GetMySentInvitationsRequest) (*pb.GetMySentInvitationsResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	var invStatus models.InvitationStatus
	if req.GetStatus() != "" {
		invStatus = models.InvitationStatus(req.GetStatus())
	}
	var bountyID *int64
	if req.GetBountyId() > 0 {
		bountyID = &req.BountyId
	}

	// 使用固定分页大小，因为 proto 没有定义 page_size
	limit := 20

	invs, err := server.store.ListInvitationsByPoster(ctx, authPayload.Username, bountyID, invStatus, limit, 0)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询邀请失败: %v", err)
	}

	return &pb.GetMySentInvitationsResponse{
		Invitations: convertInvitations(invs),
	}, nil
}

// GetMyApplications 猎人查看自己发出的申请
func (server *Server) GetMyApplications(ctx context.Context, req *pb.GetMyApplicationsRequest) (*pb.GetMyApplicationsResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	var appStatus models.ApplicationStatus
	if req.GetStatus() != "" {
		appStatus = models.ApplicationStatus(req.GetStatus())
	}

	limit := int(req.GetPageSize())
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if req.GetPageId() > 1 {
		offset = int(req.GetPageId()-1) * limit
	}

	apps, err := server.store.ListApplicationsByHunter(ctx, authPayload.Username, appStatus, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询申请失败: %v", err)
	}

	// 填充每个申请关联的悬赏信息
	result := make([]*pb.BountyApplication, len(apps))
	for i, app := range apps {
		pbApp := convertBountyApplication(&app)
		// 获取悬赏详情
		bounty, err := server.store.GetBountyByID(ctx, app.BountyID)
		if err == nil && bounty != nil {
			pbApp.Bounty = convertBounty(bounty)
		}
		result[i] = pbApp
	}

	return &pb.GetMyApplicationsResponse{
		Applications: result,
	}, nil
}

// listApplications is a helper - we reuse the store's application queries via GetBountyByID
func (server *Server) listApplications(ctx context.Context, bountyID int64) ([]models.BountyApplication, error) {
	bounty, err := server.store.GetBountyByID(ctx, bountyID)
	if err != nil {
		return nil, err
	}
	return bounty.Applications, nil
}

func convertInvitation(inv *models.Invitation) *pb.Invitation {
	pbInv := &pb.Invitation{
		Id:             inv.ID,
		BountyId:       inv.BountyID,
		PosterUsername:  inv.PosterUsername,
		HunterUsername:  inv.HunterUsername,
		Status:         string(inv.Status),
		CreatedAt:      inv.CreatedAt.Unix(),
		UpdatedAt:      inv.UpdatedAt.Unix(),
	}
	if inv.Bounty != nil {
		pbInv.Bounty = convertBounty(inv.Bounty)
	}
	return pbInv
}

func convertInvitations(invs []models.Invitation) []*pb.Invitation {
	result := make([]*pb.Invitation, len(invs))
	for i, inv := range invs {
		result[i] = convertInvitation(&inv)
	}
	return result
}
