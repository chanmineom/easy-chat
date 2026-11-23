package logic

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/im/ws/rpc/gateway"
	"github.com/chanmineom/easy-chat/apps/im/ws/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PushToUsersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPushToUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushToUsersLogic {
	return &PushToUsersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PushToUsersLogic) PushToUsers(in *gateway.PushToUsersReq) (*gateway.PushToUsersResp, error) {
	// todo: add your logic here and delete this line

	return &gateway.PushToUsersResp{}, nil
}
