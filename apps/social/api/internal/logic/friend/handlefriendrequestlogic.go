// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package friend

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/social/api/internal/auth"
	"github.com/chanmineom/easy-chat/apps/social/api/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/api/internal/types"
	"github.com/chanmineom/easy-chat/apps/social/rpc/socialclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type HandleFriendRequestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// accept or reject friend request
func NewHandleFriendRequestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HandleFriendRequestLogic {
	return &HandleFriendRequestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HandleFriendRequestLogic) HandleFriendRequest(req *types.HandleFriendRequestReq) (resp *types.HandleFriendRequestResp, err error) {
	userID, err := auth.UserID(l.ctx)
	if err != nil {
		return nil, err
	}

	response, err := l.svcCtx.Social.HandleFriendRequest(
		l.ctx,
		&socialclient.HandleFriendRequestReq{
			RequestId: req.RequestId,
			HandlerId: userID,
			Accept:    req.Accept,
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.HandleFriendRequestResp{
		FriendshipId: response.FriendshipId,
	}, nil
}
