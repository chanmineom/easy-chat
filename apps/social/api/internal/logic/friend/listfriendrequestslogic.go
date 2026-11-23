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

type ListFriendRequestsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// list incoming friend requests
func NewListFriendRequestsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFriendRequestsLogic {
	return &ListFriendRequestsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListFriendRequestsLogic) ListFriendRequests(req *types.ListFriendRequestsReq) (resp *types.ListFriendRequestsResp, err error) {
	userID, err := auth.UserID(l.ctx)
	if err != nil {
		return nil, err
	}

	response, err := l.svcCtx.Social.ListFriendRequests(
		l.ctx,
		&socialclient.ListFriendRequestsReq{
			UserId: userID,
		},
	)
	if err != nil {
		return nil, err
	}

	list := make([]*types.FriendRequestItem, 0, len(response.Requests))
	for _, item := range response.Requests {
		list = append(list, &types.FriendRequestItem{
			Id:          item.Id,
			RequesterId: item.RequesterId,
			RecipientId: item.RecipientId,
			Message:     item.Message,
			Status:      item.Status,
			CreatedAt:   item.CreatedAt,
		})
	}

	return &types.ListFriendRequestsResp{
		List: list,
	}, nil
}
