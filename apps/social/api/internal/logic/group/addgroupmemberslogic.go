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

type AddGroupMembersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// add group members
func NewAddGroupMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddGroupMembersLogic {
	return &AddGroupMembersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddGroupMembersLogic) AddGroupMembers(req *types.AddGroupMembersReq) (resp *types.AddGroupMembersResp, err error) {
	userID, err := auth.UserID(l.ctx)
	if err != nil {
		return nil, err
	}

	response, err := l.svcCtx.Social.AddGroupMembers(
		l.ctx,
		&social.AddGroupMembersReq{
			GroupId:    req.GroupId,
			OperatorId: userID,
			MemberIds:  req.MemberIds,
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.AddGroupMembersResp{
		MembershipVersion: response.MembershipVersion,
		AddedUserIds:      response.AddedUserIds,
	}, nil
}
