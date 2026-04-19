package db

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// profileProtoRequest wraps a manually-built proto message that satisfies proto.Message.
type profileProtoRequest struct {
	data []byte
}

func (m *profileProtoRequest) Reset()         { m.data = nil }
func (m *profileProtoRequest) String() string { return fmt.Sprintf("<profileProtoRequest len=%d>", len(m.data)) }
func (*profileProtoRequest) ProtoMessage()    {}
func (m *profileProtoRequest) Marshal() ([]byte, error) {
	return m.data, nil
}
func (m *profileProtoRequest) Unmarshal(_ []byte) error {
	return nil
}

// ProfileClient interface for refreshing user profile stats.
type ProfileClient interface {
	RefreshProfileStats(ctx context.Context, username string, bountyID int64, deltaCompleted int32, deltaEarnings int64, deltaPosted int32, deltaCompletedAsEmployer int32) error
}

type rawProfileClient struct {
	cc          *grpc.ClientConn
	systemToken string
}

func NewRawProfileClient(conn *grpc.ClientConn, systemToken string) ProfileClient {
	return &rawProfileClient{cc: conn, systemToken: systemToken}
}

func (c *rawProfileClient) RefreshProfileStats(ctx context.Context, username string, bountyID int64, deltaCompleted int32, deltaEarnings int64, deltaPosted int32, deltaCompletedAsEmployer int32) error {
	body, err := encodeRefreshStatsReq(username, bountyID, deltaCompleted, deltaEarnings, deltaPosted, deltaCompletedAsEmployer)
	if err != nil {
		return fmt.Errorf("编码请求失败: %w", err)
	}

	systemMD := metadata.Pairs("authorization", "Bearer "+c.systemToken)
	outgoingCtx := metadata.NewOutgoingContext(context.Background(), systemMD)

	// Pass a proto.Message wrapper for the request; nil for the response body.
	var nilResp proto.Message
	err = c.cc.Invoke(outgoingCtx, "/pb.ProfileService/RefreshProfileStats",
		&profileProtoRequest{data: body}, nilResp,
		grpc.ForceCodec(&rawCodec{}),
	)
	if err != nil {
		s, _ := status.FromError(err)
		if s.Code() == codes.NotFound || s.Code() == codes.Unavailable {
			// Profile service unavailable or user not yet created — not fatal.
			log.Printf("用户 %s 画像刷新跳过（服务不可用或用户不存在）: %v", username, err)
			return nil
		}
		log.Printf("刷新用户画像统计失败 [username=%s, bounty_id=%d]: %v", username, bountyID, err)
		return fmt.Errorf("调用 ProfileService.RefreshProfileStats 失败: %w", err)
	}
	return nil
}

// rawCodec encodes/decodes protobuf messages using raw bytes.
type rawCodec struct{}

func (c *rawCodec) Marshal(v any) ([]byte, error) {
	if req, ok := v.(*profileProtoRequest); ok {
		return req.data, nil
	}
	return nil, fmt.Errorf("rawCodec: unsupported type %T", v)
}

func (c *rawCodec) Unmarshal(_ []byte, _ any) error {
	return nil
}

func (c *rawCodec) Name() string { return "rawpb" }

// encodeRefreshStatsReq encodes RefreshProfileStatsRequest using raw protobuf wire format.
// Field tags: 1=username(Len), 2=bounty_id(Varint), 3=delta_completed(Varint), 4=delta_earnings(Varint), 5=delta_posted(Varint), 6=delta_completed_as_employer(Varint).
func encodeRefreshStatsReq(username string, bountyID int64, deltaCompleted int32, deltaEarnings int64, deltaPosted int32, deltaCompletedAsEmployer int32) ([]byte, error) {
	var buf []byte

	// Field 1: username (wire type 2 = length-delimited)
	if username != "" {
		n := len(username)
		// tag 1 << 3 | wire type 2 = 0x0a
		buf = append(buf, 0x0a)
		// length varint
		buf = appendVarint(buf, uint64(n))
		buf = append(buf, username...)
	}

	// Field 2: bounty_id (wire type 0 = varint)
	if bountyID != 0 {
		buf = append(buf, 0x10) // tag 2 << 3 | 0
		buf = appendVarint(buf, uint64(bountyID))
	}

	// Field 3: delta_completed
	if deltaCompleted != 0 {
		buf = append(buf, 0x18) // tag 3 << 3 | 0
		buf = appendVarint(buf, uint64(deltaCompleted))
	}

	// Field 4: delta_earnings
	if deltaEarnings != 0 {
		buf = append(buf, 0x20) // tag 4 << 3 | 0
		buf = appendVarint(buf, uint64(deltaEarnings))
	}

	// Field 5: delta_posted
	if deltaPosted != 0 {
		buf = append(buf, 0x28) // tag 5 << 3 | 0
		buf = appendVarint(buf, uint64(deltaPosted))
	}

	// Field 6: delta_completed_as_employer
	if deltaCompletedAsEmployer != 0 {
		buf = append(buf, 0x30) // tag 6 << 3 | 0
		buf = appendVarint(buf, uint64(deltaCompletedAsEmployer))
	}

	return buf, nil
}

// appendVarint appends a protobuf varint to buf and returns the result.
func appendVarint(buf []byte, v uint64) []byte {
	switch {
	case v < 0x80:
		return append(buf, byte(v))
	case v < 0x4000:
		return append(buf, byte(v)|0x80, byte(v>>7))
	case v < 0x200000:
		return append(buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14))
	case v < 0x10000000:
		return append(buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14)|0x80, byte(v>>21))
	case v >= 1<<63:
		// 10-byte encoding for values >= 2^63
		buf = append(buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14)|0x80, byte(v>>21)|0x80,
			byte(v>>28)|0x80, byte(v>>35)|0x80, byte(v>>42)|0x80, byte(v>>49)|0x80,
			byte(v>>56)|0x80, 1)
		return buf
	default:
		// 5-byte encoding (max value for positive int64 fits in 5 bytes)
		return append(buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14)|0x80, byte(v>>21)|0x80, byte(v>>28))
	}
}
