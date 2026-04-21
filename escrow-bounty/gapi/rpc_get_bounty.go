package gapi

import (
	"context"

	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func (server *Server) GetBounty(ctx context.Context, req *pb.GetBountyRequest) (*pb.GetBountyResponse, error) {
	_, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetId() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "非法的悬赏 ID")
	}

	bounty, err := server.store.GetBountyByID(ctx, req.GetId())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "找不到该悬赏")
		}
		return nil, status.Errorf(codes.Internal, "查询悬赏详情失败: %v", err)
	}

	// 收集所有用户名（雇主+猎人）用于批量查询头像
	usernameSet := make(map[string]struct{})
	usernameSet[bounty.EmployerUsername] = struct{}{}
	for _, app := range bounty.Applications {
		usernameSet[app.HunterUsername] = struct{}{}
	}
	usernames := make([]string, 0, len(usernameSet))
	for u := range usernameSet {
		usernames = append(usernames, u)
	}
	avatarMap, _ := server.store.GetAvatarUrlsByUsernames(ctx, usernames)

	pbBounty := convertBounty(bounty)
	pbBounty.EmployerAvatarUrl = avatarMap[bounty.EmployerUsername]

	// 转换关联的申请列表数据
	var pbApplications []*pb.BountyApplication
	for _, app := range bounty.Applications {
		currentApp := app
		pbApp := convertBountyApplication(&currentApp)
		pbApp.HunterAvatarUrl = avatarMap[app.HunterUsername]
		pbApplications = append(pbApplications, pbApp)
	}

	return &pb.GetBountyResponse{
		Bounty:       pbBounty,
		Applications: pbApplications,
	}, nil
}
