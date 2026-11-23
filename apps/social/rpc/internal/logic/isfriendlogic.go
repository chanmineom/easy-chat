package logic

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type IsFriendLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewIsFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IsFriendLogic {
	return &IsFriendLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *IsFriendLogic) IsFriend(in *social.IsFriendReq) (*social.IsFriendResp, error) {
	isFriend, err := l.svcCtx.Store.IsFriend(
		l.ctx,
		in.UserA,
		in.UserB,
	)
	if err != nil {
		return nil, status.Error(
			codes.Internal,
			"could not check friendship",
		)
	}

	return &social.IsFriendResp{
		IsFriend: isFriend,
	}, nil
}
