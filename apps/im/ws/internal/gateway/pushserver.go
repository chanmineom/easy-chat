package gateway

import (
	"context"

	gatewaypb "github.com/chanmineom/easy-chat/apps/im/ws/rpc/gateway"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MessageCreatedData struct {
	MessageID       string `json:"message_id"`
	ConversationID  string `json:"conversation_id"`
	Sequence        int64  `json:"sequence"`
	SenderID        string `json:"sender_id"`
	ClientMessageID string `json:"client_message_id"`
	ContentType     string `json:"content_type"`
	Content         string `json:"content"`
	CreatedAt       int64  `json:"created_at"`
}

type MessageRejectedData struct {
	CommandID       string `json:"command_id"`
	ConversationID  string `json:"conversation_id"`
	ClientMessageID string `json:"client_message_id"`
	Code            string `json:"code"`
	Message         string `json:"message"`
}

type PushServer struct {
	gatewaypb.UnimplementedGatewayServer
	hub *Hub
}

func NewPushServer(hub *Hub) *PushServer {
	return &PushServer{
		hub: hub,
	}
}

func (s *PushServer) PushToUsers(_ context.Context, in *gatewaypb.PushToUsersReq) (*gatewaypb.PushToUsersResp, error) {
	if in == nil || in.Message == nil {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}
	if len(in.UserIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "user ids are required")
	}

	envelop, err := NewDataEnvelope(TypeMessageCreated, MessageCreatedData{
		MessageID:       in.Message.MessageId,
		ConversationID:  in.Message.ConversationId,
		Sequence:        in.Message.Sequence,
		SenderID:        in.Message.SenderId,
		ClientMessageID: in.Message.ClientMessageId,
		ContentType:     in.Message.ContentType,
		Content:         in.Message.Content,
		CreatedAt:       in.Message.CreatedAt,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "encode message event")
	}

	onlineUsers, deliveredConnections := s.hub.SendToUsers(in.UserIds, envelop)

	return &gatewaypb.PushToUsersResp{
		OnlineUsers:          int32(onlineUsers),
		DeliveredConnections: int32(deliveredConnections),
	}, nil
}

func (s *PushServer) PushCommandResult(_ context.Context, in *gatewaypb.PushCommandResultReq) (*gatewaypb.PushCommandResultResp, error) {
	if in == nil || in.Result == nil {
		return nil, status.Error(codes.InvalidArgument, "command result is required")
	}
	if in.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	envelop, err := NewDataEnvelope(TypeMessageRejected, MessageRejectedData{
		CommandID: in.Result.CommandId,
		ConversationID: in.Result.ConversationId,
		ClientMessageID: in.Result.ClientMessageId,
		Code: in.Result.Code,
		Message: in.Result.Message,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "encode command result")
	}

	onlineUsers, _ := s.hub.SendToUsers([]string{in.UserId}, envelop)

	_ = onlineUsers

	return &gatewaypb.PushCommandResultResp{
		// 当前响应没有字段，保留扩展位置。
	}, nil
}
