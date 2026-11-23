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

type LeaveGroupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// leave group
func NewLeaveGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LeaveGroupLogic {
	return &LeaveGroupLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LeaveGroupLogic) LeaveGroup(req *types.LeaveGroupReq) (resp *types.LeaveGroupResp, err error) {
	userID, err := auth.UserID(l.ctx)
	if err != nil {
		return nil, err
	}

	response, err := l.svcCtx.Social.LeaveGroup(
		l.ctx,
		&social.LeaveGroupReq{
			GroupId: req.GroupId,
			UserId:  userID,
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.LeaveGroupResp{
		MembershipVersion: response.MembershipVersion,
	}, nil
}
