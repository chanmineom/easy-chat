package logic

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"

	"github.com/zeromicro/go-zero/core/logx"
)

type HandleFriendRequestLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewHandleFriendRequestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HandleFriendRequestLogic {
	return &HandleFriendRequestLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *HandleFriendRequestLogic) HandleFriendRequest(in *social.HandleFriendRequestReq) (*social.HandleFriendRequestResp, error) {
	result, err := l.svcCtx.Store.HandleFriendRequest(
		l.ctx,
		in.RequestId,
		in.HandlerId,
		in.Accept,
	)
	if err != nil {
		return nil, mapFriendError(l.Logger, err)
	}

	return &social.HandleFriendRequestResp{
		FriendshipId: result.FriendshipID,
	}, nil
}
