package logic

import (
	"context"
	"errors"
	"time"

	"github.com/chanmineom/easy-chat/apps/user/auth"
	"github.com/chanmineom/easy-chat/apps/user/repository"
	"github.com/chanmineom/easy-chat/apps/user/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/user/rpc/user"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginLogic) Login(in *user.LoginReq) (*user.LoginResp, error) {
	if in == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"request is required",
		)
	}

	phone, err := normalizePhone(in.Phone)
	if err != nil {
		return nil, status.Error(
			codes.Unauthenticated,
			"invalid credentials",
		)
	}

	record, err := l.svcCtx.Store.FindByPhone(l.ctx, phone)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Error(
				codes.Unauthenticated,
				"invalid credentials",
			)
		}

		l.Errorf("find user: %v", err)
		return nil, status.Error(
			codes.Internal,
			"could not log in",
		)
	}

	if record.Status != 1 {
		return nil, status.Error(
			codes.PermissionDenied,
			"user is disabled",
		)
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(record.PasswordHash),
		[]byte(in.Password),
	); err != nil {
		return nil, status.Error(
			codes.Unauthenticated,
			"invalid credentials",
		)
	}

	now := time.Now().UTC()
	token, err := auth.GenerateToken(
		l.svcCtx.Config.Jwt.AccessSecret,
		record.ID,
		l.svcCtx.Config.Jwt.AccessExpire,
		now,
	)
	if err != nil {
		l.Errorf("generate token: %v", err)
		return nil, status.Error(
			codes.Internal,
			"could not log in",
		)
	}

	return &user.LoginResp{
		Token:  token.Token,
		Expire: token.Expire,
		UserId: record.ID,
	}, nil
}
