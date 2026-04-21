package gapi

import (
	"context"

	"github.com/grayfalcon666/escrow-bounty/models"
	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) ListBounties(ctx context.Context, req *pb.ListBountiesRequest) (*pb.ListBountiesResponse, error) {
	// 悬赏大厅公开访问，不需要登录

	limit := int(req.GetPageSize())
	if limit <= 0 || limit > 100 {
		limit = 20 // 默认每页 20 条
	}

	offset := int((req.GetPageId() - 1) * req.GetPageSize())
	if offset < 0 {
		offset = 0
	}

	bounties, err := server.store.ListBounties(ctx, models.BountyStatus(req.GetStatus()), limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询悬赏列表失败: %v", err)
	}

	// 批量获取雇主头像
	employerSet := make(map[string]struct{})
	for _, b := range bounties {
		employerSet[b.EmployerUsername] = struct{}{}
	}
	employerUsernames := make([]string, 0, len(employerSet))
	for u := range employerSet {
		employerUsernames = append(employerUsernames, u)
	}
	employerAvatarMap, _ := server.store.GetAvatarUrlsByUsernames(ctx, employerUsernames)

	var pbBounties []*pb.Bounty
	for _, bounty := range bounties {
		pbB := convertBounty(&bounty)
		pbB.EmployerAvatarUrl = employerAvatarMap[bounty.EmployerUsername]
		pbBounties = append(pbBounties, pbB)
	}

	return &pb.ListBountiesResponse{
		Bounties: pbBounties,
	}, nil
}
