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

type KickGroupMemberLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// kick group member
func NewKickGroupMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KickGroupMemberLogic {
	return &KickGroupMemberLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *KickGroupMemberLogic) KickGroupMember(req *types.KickGroupMemberReq) (resp *types.KickGroupMemberResp, err error) {
	userID, err := auth.UserID(l.ctx)
	if err != nil {
		return nil, err
	}

	response, err := l.svcCtx.Social.KickGroupMember(
		l.ctx,
		&socialclient.KickGroupMemberReq{
			GroupId:    req.GroupId,
			OperatorId: userID,
			UserId:     req.UserId,
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.KickGroupMemberResp{
		MembershipVersion: response.MembershipVersion,
	}, nil
}
