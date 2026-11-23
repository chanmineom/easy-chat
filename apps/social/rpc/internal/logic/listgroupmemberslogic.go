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

type ListGroupMembersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListGroupMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListGroupMembersLogic {
	return &ListGroupMembersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListGroupMembersLogic) ListGroupMembers(in *social.ListGroupMembersReq) (*social.ListGroupMembersResp, error) {
	if in == nil || in.UserId == "" || in.GroupId == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"user and group are required",
		)
	}

	rows, err := l.svcCtx.Store.ListActiveGroupMembers(
		l.ctx,
		in.UserId,
		in.GroupId,
	)
	if err != nil {
		if errors.Is(err, repository.ErrGroupPermissionDenied) {
			return nil, status.Error(
				codes.PermissionDenied,
				"user is not an active group member",
			)
		}

		return nil, status.Error(
			codes.Internal,
			"could not load group members",
		)
	}

	members := make([]*social.GroupMemberItem, 0, len(rows))
	for _, row := range rows {
		members = append(members, &social.GroupMemberItem{
			UserId:   row.UserID,
			Role:     int32(row.Role),
			Status:   int32(row.Status),
			JoinedAt: row.JoinedAt.UnixMilli(),
		})
	}

	return &social.ListGroupMembersResp{
		Members: members,
	}, nil
}
