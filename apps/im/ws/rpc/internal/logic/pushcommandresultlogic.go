package logic

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/im/ws/rpc/gateway"
	"github.com/chanmineom/easy-chat/apps/im/ws/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PushCommandResultLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPushCommandResultLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushCommandResultLogic {
	return &PushCommandResultLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PushCommandResultLogic) PushCommandResult(in *gateway.PushCommandResultReq) (*gateway.PushCommandResultResp, error) {
	// todo: add your logic here and delete this line

	return &gateway.PushCommandResultResp{}, nil
}
