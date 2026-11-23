package logic

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"github.com/chanmineom/easy-chat/apps/im/rpc/im"
	"github.com/chanmineom/easy-chat/apps/im/rpc/internal/svc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetUpUserConversationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSetUpUserConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetUpUserConversationLogic {
	return &SetUpUserConversationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SetUpUserConversationLogic) SetUpUserConversation(in *im.SetUpUserConversationReq) (*im.SetUpUserConversationResp, error) {
	// todo: add your logic here and delete this line
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	switch model.ChatType(in.ChatType) {
	case model.ChatTypeSingle:
		conversation, err := l.svcCtx.Mongo.EnsureDirectConversation(l.ctx, in.SendId, in.RecvId)
		if err != nil {
			l.Errorf("ensure direct conversation send=%s recv=%s: %v", in.SendId, in.RecvId, err)
			return nil, status.Error(codes.Internal, "could not create direct conversation")
		}
		return &im.SetUpUserConversationResp{
			ConversationId: conversation.ID,
			ChatType:       int32(conversation.ChatType),
		}, nil

	case model.ChatTypeGroup:
		return nil, status.Errorf(codes.Unimplemented, "group conversation setup is not implemented yet")

	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported chat type")
	}
}
