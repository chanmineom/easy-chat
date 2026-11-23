package logic

import (
	"context"
	"errors"
	"time"

	"github.com/chanmineom/easy-chat/apps/user/auth"
	"github.com/chanmineom/easy-chat/apps/user/model"
	"github.com/chanmineom/easy-chat/apps/user/repository"
	"github.com/chanmineom/easy-chat/apps/user/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/user/rpc/user"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterLogic) Register(in *user.RegisterReq) (*user.RegisterResp, error) {
	if in == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"request is required",
		)
	}

	phone, err := normalizePhone(in.Phone)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := validatePassword(in.Password); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	nickname, err := normalizeNickname(in.Nickname)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	avatarURL, err := validateAvatarURL(in.AvatarUrl)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(in.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		l.Errorf("hash password: %v", err)
		return nil, status.Error(
			codes.Internal,
			"could not create user",
		)
	}

	userID, err := model.NewUserID()
	if err != nil {
		return nil, status.Error(
			codes.Internal,
			"could not create user",
		)
	}

	now := time.Now().UTC()
	record := model.User{
		ID:           userID,
		Phone:        phone,
		PasswordHash: string(passwordHash),
		Nickname:     nickname,
		AvatarURL:    avatarURL,
		Status:       1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := l.svcCtx.Store.CreateUser(l.ctx, record); err != nil {
		if errors.Is(err, repository.ErrPhoneRegistered) {
			return nil, status.Error(
				codes.AlreadyExists,
				"phone is already registered",
			)
		}

		l.Errorf("create user: %v", err)
		return nil, status.Errorf(
			codes.Internal,
			"could not create user",
		)
	}

	token, err := auth.GenerateToken(
		l.svcCtx.Config.Jwt.AccessSecret,
		userID,
		l.svcCtx.Config.Jwt.AccessExpire,
		now,
	)
	if err != nil {
		l.Errorf("generate token: %v", err)
		return nil, status.Error(
			codes.Internal,
			"user created but token generation failed",
		)
	}

	return &user.RegisterResp{
		Token:  token.Token,
		Expire: token.Expire,
		UserId: userID,
	}, nil
}
