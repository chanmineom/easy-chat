package logic

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListGroupsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListGroupsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListGroupsLogic {
	return &ListGroupsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListGroupsLogic) ListGroups(in *social.ListGroupsReq) (*social.ListGroupResp, error) {
	if in == nil || in.UserId == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"user id is required",
		)
	}

	rows, err := l.svcCtx.Store.ListUserGroups(l.ctx, in.UserId)
	if err != nil {
		l.Error("list groups: %v", err)
		return nil, status.Error(
			codes.Internal,
			"could not load groups",
		)
	}

	groups := make([]*social.GroupSummary, 0, len(rows))
	for _, row := range rows {
		groups = append(groups, &social.GroupSummary{
			GroupId:      row.ID,
			Name:         row.Name,
			AvatarUrl:    row.AvatarURL,
			Notification: row.Notification,
			OwnerId:      row.OwnerID,
			Role:         int32(row.Role),
			Status:       int32(row.Status),
			CreatedAt:    row.CreatedAt.UnixMilli(),
			UpdatedAt:    row.UpdatedAt.UnixMilli(),
		})
	}

	return &social.ListGroupResp{
		Groups: groups,
	}, nil
}
