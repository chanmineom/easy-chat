// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package user

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/user/api/internal/svc"
	"github.com/chanmineom/easy-chat/apps/user/api/internal/types"
	"github.com/chanmineom/easy-chat/apps/user/rpc/userclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// register
func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterReq) (resp *types.RegisterResp, err error) {
	response, err := l.svcCtx.User.Register(
		l.ctx,
		&userclient.RegisterReq{
			Phone:     req.Phone,
			Password:  req.Password,
			Nickname:  req.Nickname,
			AvatarUrl: req.AvatarUrl,
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.RegisterResp{
		Token:  response.Token,
		Expire: response.Expire,
		UserId: response.UserId,
	}, nil
}
