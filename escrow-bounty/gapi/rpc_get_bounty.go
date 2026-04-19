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

	pbBounty := convertBounty(bounty)

	// 转换关联的申请列表数据
	var pbApplications []*pb.BountyApplication
	for _, app := range bounty.Applications {
		currentApp := app
		pbApplications = append(pbApplications, convertBountyApplication(&currentApp))
	}

	return &pb.GetBountyResponse{
		Bounty:       pbBounty,
		Applications: pbApplications,
	}, nil
}
