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

type ChangeGroupMemberRoleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// change member role
func NewChangeGroupMemberRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangeGroupMemberRoleLogic {
	return &ChangeGroupMemberRoleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChangeGroupMemberRoleLogic) ChangeGroupMemberRole(req *types.ChangeGroupMemberRoleReq) (resp *types.ChangeGroupMemberRoleResp, err error) {
	userID, err := auth.UserID(l.ctx)
	if err != nil {
		return nil, err
	}

	response, err := l.svcCtx.Social.ChangeGroupMemberRole(
		l.ctx,
		&social.ChangeGroupMemberRoleReq{
			GroupId:    req.GroupId,
			OperatorId: userID,
			UserId:     req.UserId,
			Role:       req.Role,
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.ChangeGroupMemberRoleResp{
		MembershipVersion: response.MembershipVersion,
	}, nil
}
