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

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// login
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	response, err := l.svcCtx.User.Login(
		l.ctx,
		&userclient.LoginReq{
			Phone:    req.Phone,
			Password: req.Password,
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.LoginResp{
		Token:  response.Token,
		Expire: response.Expire,
		UserId: response.UserId,
	}, nil
}
