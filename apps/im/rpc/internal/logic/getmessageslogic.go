package logic

import (
	"context"
	"errors"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"github.com/chanmineom/easy-chat/apps/im/repository"
	"github.com/chanmineom/easy-chat/apps/im/rpc/im"
	"github.com/chanmineom/easy-chat/apps/im/rpc/internal/svc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMessagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMessagesLogic {
	return &GetMessagesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetMessagesLogic) GetMessages(in *im.GetMessagesReq) (*im.GetMessagesResp, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if in.UserId == "" {
		return nil, status.Error(codes.Unauthenticated, "user id is required")
	}
	if in.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation id is required")
	}
	if in.AfterSeq < 0 {
		return nil, status.Error(codes.InvalidArgument, "after_seq must not be negative")
	}
	if in.BeforeReq < 0 {
		return nil, status.Error(codes.InvalidArgument, "before_seq must not be negative")
	}
	if in.AfterSeq > 0 && in.BeforeReq > 0 {
		return nil, status.Error(codes.InvalidArgument, "after_seq and before_seq cannot be used together")
	}

	limit := int(in.Limit)
	if limit <= 0 {
		limit = 50
	}

	// messages, hasMore, err := l.svcCtx.Mongo.ListMessagesAfter(
	// 	l.ctx,
	// 	in.UserId,
	// 	in.ConversationId,
	// 	in.AfterSeq,
	// 	limit,
	// )
	var (
		messages []model.Message
		hasMore  bool
		err      error
	)

	if in.BeforeReq > 0 {
		messages, hasMore, err = l.svcCtx.Mongo.ListMessagesBefore(
			l.ctx, in.UserId, in.ConversationId, in.BeforeReq, limit,
		)
	} else {
		messages, hasMore, err = l.svcCtx.Mongo.ListMessagesAfter(
			l.ctx, in.UserId, in.ConversationId, in.AfterSeq, limit,
		)
	}
	if err != nil {
		if errors.Is(err, repository.ErrConversationAccessDenied) {
			return nil, status.Error(codes.PermissionDenied, "user is not a conversation member")
		}
		l.Errorf("list messages: %v", err)
		return nil, status.Error(codes.Internal, "could not load messages")
	}

	result := make([]*im.ChatMessage, 0, len(messages))
	for _, message := range messages {
		result = append(result, &im.ChatMessage{
			MessageId:       message.ID.Hex(),
			ConversationId:  message.ConversationID,
			Sequence:        message.Sequence,
			SenderId:        message.SenderID,
			ClientMessageId: message.ClientMessageID,
			ContentType:     message.ContentType,
			Content:         message.Content,
			CreatedAt:       message.CreatedAt.UnixMilli(),
		})
	}

	return &im.GetMessagesResp{
		Messages: result,
		HasMore:  hasMore,
	}, nil
}
