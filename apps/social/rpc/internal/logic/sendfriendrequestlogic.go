package logic

import (
	"context"
	"errors"

	"github.com/chanmineom/easy-chat/apps/social/repository"
	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendFriendRequestLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendFriendRequestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendFriendRequestLogic {
	return &SendFriendRequestLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendFriendRequestLogic) SendFriendRequest(in *social.SendFriendRequestReq) (*social.SendFriendRequestResp, error) {

	
	result, err := l.svcCtx.Store.SendFriendRequest(
		l.ctx,
		in.RequesterId,
		in.RecipientId,
		in.Message,
	)
	if err != nil {
		return nil, mapFriendError(l.Logger, err)
	}

	return &social.SendFriendRequestResp{
		RequestId: result.RequestID,
	}, nil
}

func mapFriendError(logger logx.Logger, err error) error {
	switch {
	case errors.Is(err, repository.ErrCannotFriendSelf):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, repository.ErrAlreadyFriends),
		errors.Is(err, repository.ErrFriendRequestExists):
		return status.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, repository.ErrFriendRequestNotFound),
		errors.Is(err, repository.ErrFriendshipNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, repository.ErrFriendRequestForbidden):
		return status.Error(codes.PermissionDenied, err.Error())

	case errors.Is(err, repository.ErrFriendRequestHandled):
		return status.Error(codes.FailedPrecondition, err.Error())

	default:
		logger.Errorf("friend operation failed: %v", err)
		return status.Error(codes.Internal, "friend operation failed")
	}
}