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

type DissolveGroupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// dissolve group
func NewDissolveGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DissolveGroupLogic {
	return &DissolveGroupLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DissolveGroupLogic) DissolveGroup(req *types.DissolveGroupReq) (resp *types.DissolveGroupResp, err error) {
	userID, err := auth.UserID(l.ctx)
	if err != nil {
		return nil, err
	}

	response, err := l.svcCtx.Social.DissolveGroup(
		l.ctx,
		&social.DissolveGroupReq{
			GroupId:    req.GroupId,
			OperatorId: userID,
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.DissolveGroupResp{
		MembershipVersion: response.MembershipVersion,
	}, nil
}
