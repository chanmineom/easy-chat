package logic

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListFriendRequestsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListFriendRequestsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFriendRequestsLogic {
	return &ListFriendRequestsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListFriendRequestsLogic) ListFriendRequests(in *social.ListFriendRequestsReq) (*social.ListFriendRequestsResp, error) {
	if in == nil || in.UserId == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"user id is required",
		)
	}

	requests, err := l.svcCtx.Store.ListIncomingFriendRequests(
		l.ctx,
		in.UserId,
	)
	if err != nil {
		l.Errorf("list friend requests: %v", err)
		return nil, status.Error(
			codes.Internal,
			"could not load friend requests",
		)
	}

	items := make([]*social.FriendRequestItem, 0, len(requests))
	for _, request := range requests {
		items = append(items, &social.FriendRequestItem{
			Id:          request.ID,
			RequesterId: request.RequesterID,
			RecipientId: request.RecipientID,
			Message:     request.Message,
			Status:      int32(request.Status),
			CreatedAt:   request.CreatedAt.UnixMilli(),
		})
	}

	return &social.ListFriendRequestsResp{
		Requests: items,
	}, nil
}
