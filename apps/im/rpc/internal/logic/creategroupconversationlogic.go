package logic

import (
	"context"
	"errors"

	"github.com/chanmineom/easy-chat/apps/im/repository"
	"github.com/chanmineom/easy-chat/apps/im/rpc/im"
	"github.com/chanmineom/easy-chat/apps/im/rpc/internal/svc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateGroupConversationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateGroupConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateGroupConversationLogic {
	return &CreateGroupConversationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateGroupConversationLogic) CreateGroupConversation(in *im.CreateGroupConversationReq) (*im.CreateGroupConversationResp, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	conversation, err := l.svcCtx.Mongo.EnsureGroupConversation(
		l.ctx,
		repository.EnsureGroupConversationInput{
			GroupID:   in.GroupId,
			CreatorID: in.CreatorId,
			MemberIDs: in.MemberIds,
			MembershipVersion: uint64(in.MembershipVersion),
		},
	)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidGroupConversation) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, repository.ErrConversationTypeConflict) {
			return nil, status.Error(codes.AlreadyExists, "conversation type conflicts with existing conversation")
		}

		l.Errorf("create group conversation: %v", err)
		return nil, status.Error(codes.Internal, "could not create group conversation")
	}

	return &im.CreateGroupConversationResp{
		ConversationId: conversation.ID,
		ChatType:       int32(conversation.ChatType),
	}, nil
}
