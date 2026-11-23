// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package user

import (
	"context"
	"errors"

	"github.com/chanmineom/easy-chat/apps/user/api/internal/svc"
	"github.com/chanmineom/easy-chat/apps/user/api/internal/types"
	"github.com/chanmineom/easy-chat/apps/user/rpc/userclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type DetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// get current user
func NewDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DetailLogic {
	return &DetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DetailLogic) Detail(req *types.UserInfoReq) (resp *types.UserInfoResp, err error) {
	userID, ok := l.ctx.Value("easy-chat-uid").(string)
	if !ok || userID == "" {
		return nil, errors.New("authenticated user id is missing")
	}

	response, err := l.svcCtx.User.GetUserInfo(
		l.ctx,
		&userclient.GetUserInfoReq{
			UserId: userID,
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.UserInfoResp{
		Info: types.UserInfo{
			Id:        response.User.Id,
			Phone:     response.User.Phone,
			Nickname:  response.User.Nickname,
			AvatarUrl: response.User.AvatarUrl,
			Status:    response.User.Status,
		},
	}, nil
}
