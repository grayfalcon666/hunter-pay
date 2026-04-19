package gapi

import (
	db "simplebank/db/sqlc"
	"simplebank/pb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func convertUser(user db.User) *pb.User {
	return &pb.User{
		Username: user.Username,
		FullName: user.FullName,
		Email:    user.Email,
		// Timestamppb 是 Google 提供的工具，把 time.Time 转成 proto Timestamp
		PasswordChangedAt: timestamppb.New(user.PasswordChangedAt),
		CreatedAt:         timestamppb.New(user.CreatedAt),
	}
}
