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

type RemoveFriendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// remove friend
func NewRemoveFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveFriendLogic {
	return &RemoveFriendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveFriendLogic) RemoveFriend(req *types.RemoveFriendReq) (resp *types.RemoveFriendResp, err error) {
	userID, err := auth.UserID(l.ctx)
	if err != nil {
		return nil, err
	}

	_, err = l.svcCtx.Social.RemoveFriend(
		l.ctx,
		&socialclient.RemoveFriendReq{
			UserId:   userID,
			FriendId: req.FriendId,
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.RemoveFriendResp{}, nil
}
