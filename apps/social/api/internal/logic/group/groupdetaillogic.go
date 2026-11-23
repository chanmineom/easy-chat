// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package group

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/social/api/internal/auth"
	"github.com/chanmineom/easy-chat/apps/social/api/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/api/internal/types"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"

	"github.com/zeromicro/go-zero/core/logx"
)

type GroupDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// get group
func NewGroupDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupDetailLogic {
	return &GroupDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupDetailLogic) GroupDetail(req *types.GroupDetailReq) (resp *types.GroupDetailResp, err error) {
	userID, err := auth.UserID(l.ctx)
	if err != nil {
		return nil, err
	}

	response, err := l.svcCtx.Social.GetGroup(
		l.ctx,
		&social.GetGroupReq{
			UserId:  userID,
			GroupId: req.GroupId,
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.GroupDetailResp{
		Group: types.GroupSummary{
			GroupId:      response.Group.GroupId,
			Name:         response.Group.Name,
			AvatarUrl:    response.Group.AvatarUrl,
			Notification: response.Group.Notification,
			OwnerId:      response.Group.OwnerId,
			Role:         response.Group.Role,
			Status:       response.Group.Status,
			CreatedAt:    response.Group.CreatedAt,
			UpdatedAt:    response.Group.UpdatedAt,
		},
	}, nil
}
