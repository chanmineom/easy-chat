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

type AddGroupMembersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddGroupMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddGroupMembersLogic {
	return &AddGroupMembersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AddGroupMembersLogic) AddGroupMembers(in *social.AddGroupMembersReq) (*social.AddGroupMembersResp, error) {
	if in == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"request is required",
		)
	}

	result, err := l.svcCtx.Store.AddGroupMembers(
		l.ctx,
		in.GroupId,
		in.OperatorId,
		in.MemberIds,
	)
	if err != nil {
		return nil, mapGroupMutationError(l.Logger, err)
	}

	return &social.AddGroupMembersResp{
		MembershipVersion: int64(result.MembershipVersion),
		AddedUserIds:      result.AddedUserIDs,
	}, nil
}

func mapGroupMutationError(
	logger logx.Logger,
	err error,
) error {
	switch {
	case errors.Is(err, repository.ErrInvalidGroup):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, repository.ErrGroupNotFound),
		errors.Is(err, repository.ErrGroupNotActive),
		errors.Is(err, repository.ErrGroupMemberNotFound),
		errors.Is(err, repository.ErrGroupMemberNotActive):
		return status.Error(codes.FailedPrecondition, err.Error())

	case errors.Is(err, repository.ErrGroupPermissionDenied),
		errors.Is(err, repository.ErrGroupOwnerCannotBeKicked):
		return status.Error(codes.PermissionDenied, err.Error())

	case errors.Is(err, repository.ErrGroupTooLarge):
		return status.Error(codes.ResourceExhausted, err.Error())

	default:
		logger.Errorf("group mutation failed: %v", err)
		return status.Error(codes.Internal, "group mutation failed")
	}
}
