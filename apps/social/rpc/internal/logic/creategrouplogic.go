package logic

import (
	"context"
	"errors"

	"github.com/chanmineom/easy-chat/apps/social/repository"
	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateGroupLogic {
	return &CreateGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateGroupLogic) CreateGroup(in *social.CreateGroupReq) (*social.CreateGroupResp, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	group, err := l.svcCtx.Store.CreateGroup(
		l.ctx,
		repository.CreateGroupInput{
			CreatorID:    in.CreatorId,
			Name:         in.Name,
			AvatarURL:    in.AvatarUrl,
			Notification: in.Notification,
			MemberIDs:    in.MemberIds,
		},
	)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidGroup) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		l.Errorf("create group: %v", err)
		return nil, status.Error(codes.Internal, "could not create group")
	}

	return &social.CreateGroupResp{
		GroupId: group.GroupID,
	}, nil
}
