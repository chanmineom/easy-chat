package logic

import (
	"context"
	"errors"

	"github.com/chanmineom/easy-chat/apps/user/repository"
	"github.com/chanmineom/easy-chat/apps/user/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/user/rpc/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUserInfoLogic) GetUserInfo(in *user.GetUserInfoReq) (*user.GetUserInfoResp, error) {
	if in == nil || in.UserId == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"user id is required",
		)
	}

	record, err := l.svcCtx.Store.FindByID(l.ctx, in.UserId)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Error(
				codes.NotFound,
				"user not found",
			)
		}

		return nil, status.Error(
			codes.Internal,
			"could not load user",
		)
	}

	return &user.GetUserInfoResp{
		User: &user.UserEntity{
			Id:        record.ID,
			Phone:     record.Phone,
			Nickname:  record.Nickname,
			AvatarUrl: record.AvatarURL,
			Status:    int32(record.Status),
		},
	}, nil
}
