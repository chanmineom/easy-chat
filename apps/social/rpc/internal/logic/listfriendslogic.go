package logic

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListFriendsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListFriendsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFriendsLogic {
	return &ListFriendsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListFriendsLogic) ListFriends(in *social.ListFriendsReq) (*social.ListFriendsResp, error) {
	friendIDs, err := l.svcCtx.Store.ListFriendIDs(l.ctx, in.UserId)
	if err != nil {
		return nil, status.Error(
			codes.Internal,
			"could not load friends",
		)
	}

	return &social.ListFriendsResp{
		FriendIds: friendIDs,
	}, nil
}
