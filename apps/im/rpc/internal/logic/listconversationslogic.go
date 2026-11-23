package logic

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/im/rpc/im"
	"github.com/chanmineom/easy-chat/apps/im/rpc/internal/svc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListConversationsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListConversationsLogic {
	return &ListConversationsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListConversationsLogic) ListConversations(in *im.ListConversationsReq) (*im.ListConversationsResp, error) {
	if in == nil || in.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	summaries, hasMore, err := l.svcCtx.Mongo.ListConversationSummaries(
		l.ctx,
		in.UserId,
		int(in.Limit),
		in.BeforeActivityAt,
		in.BeforeConversationId,
	)
	if err != nil {
		l.Errorf("list conversations: %v", err)
		return nil, status.Error(
			codes.Internal,
			"could not load conversations",
		)
	}

	result := make([]*im.ConversationSummary, 0, len(summaries))
	for _, summary := range summaries {
		var lastMessage *im.ChatMessage
		if summary.LastMessage != nil {
			lastMessage = &im.ChatMessage{
				MessageId:       summary.LastMessage.ID.Hex(),
				ConversationId:  summary.LastMessage.ConversationID,
				Sequence:        summary.LastMessage.Sequence,
				SenderId:        summary.LastMessage.SenderID,
				ClientMessageId: summary.LastMessage.ClientMessageID,
				ContentType:     summary.LastMessage.ContentType,
				Content:         summary.LastMessage.Content,
				CreatedAt:       summary.LastMessage.CreatedAt.UnixMilli(),
			}
		}

		result = append(result, &im.ConversationSummary{
			ConversationId: summary.ConversationID,
			ChatType:       int32(summary.ChatType),
			LastSeq:        summary.LastSeq,
			ReadSeq:        summary.ReadSeq,
			DeliveredSeq:   summary.DeliveredSeq,
			UnreadCount:    summary.UnreadCount,
			LastActivityAt: summary.LastActivityAt.UnixMilli(),
			LastMessage:    lastMessage,
		})
	}

	return &im.ListConversationsResp{
		Conversations: result,
		HasMore:       hasMore,
	}, nil
}
