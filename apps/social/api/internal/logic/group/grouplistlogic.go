// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package group

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/social/api/internal/auth"
	"github.com/chanmineom/easy-chat/apps/social/api/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/api/internal/types"
	"github.com/chanmineom/easy-chat/apps/social/rpc/socialclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type GroupListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// list my groups
func NewGroupListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupListLogic {
	return &GroupListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupListLogic) GroupList(req *types.GroupListReq) (resp *types.GroupListResp, err error) {
	userID, err := auth.UserID(l.ctx)
	if err != nil {
		return nil, err
	}

	response, err := l.svcCtx.Social.ListGroups(
		l.ctx,
		&socialclient.ListGroupsReq{
			UserId: userID,
		},
	)
	if err != nil {
		return nil, err
	}

	list := make([]*types.GroupSummary, 0, len(response.Groups))
	for _, group := range response.Groups {
		list = append(list, &types.GroupSummary{
			GroupId:      group.GroupId,
			Name:         group.Name,
			AvatarUrl:    group.AvatarUrl,
			Notification: group.Notification,
			OwnerId:      group.OwnerId,
			Role:         group.Role,
			Status:       group.Status,
			CreatedAt:    group.CreatedAt,
			UpdatedAt:    group.UpdatedAt,
		})
	}

	return &types.GroupListResp{
		List: list,
	}, nil
}
