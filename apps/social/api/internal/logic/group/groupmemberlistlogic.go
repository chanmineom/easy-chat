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

type GroupMemberListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// lsit group members
func NewGroupMemberListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupMemberListLogic {
	return &GroupMemberListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupMemberListLogic) GroupMemberList(req *types.GroupMemberListReq) (resp *types.GroupMemberListResp, err error) {
	userID, err := auth.UserID(l.ctx)
	if err != nil {
		return nil, err
	}

	response, err := l.svcCtx.Social.ListGroupMembers(
		l.ctx,
		&socialclient.ListGroupMembersReq{
			UserId:  userID,
			GroupId: req.GroupId,
		},
	)
	if err != nil {
		return nil, err
	}

	list := make([]*types.GroupMemberItem, 0, len(response.Members))
	for _, member := range response.Members {
		list = append(list, &types.GroupMemberItem{
			UserId:   member.UserId,
			Role:     member.Role,
			Status:   member.Status,
			JoinedAt: member.JoinedAt,
		})
	}

	return &types.GroupMemberListResp{
		List: list,
	}, nil
}
