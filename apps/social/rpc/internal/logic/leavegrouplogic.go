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

type LeaveGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLeaveGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LeaveGroupLogic {
	return &LeaveGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LeaveGroupLogic) LeaveGroup(in *social.LeaveGroupReq) (*social.LeaveGroupResp, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	result, err := l.svcCtx.Store.LeaveGroup(
		l.ctx,
		in.GroupId,
		in.UserId,
	)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrInvalidGroup),
			errors.Is(err, repository.ErrGroupNotFound),
			errors.Is(err, repository.ErrGroupNotActive),
			errors.Is(err, repository.ErrGroupMemberNotFound),
			errors.Is(err, repository.ErrGroupMemberNotActive):
			return nil, status.Error(
				codes.FailedPrecondition,
				err.Error(),
			)

		case errors.Is(err, repository.ErrGroupOwnerCannotLeave):
			return nil, status.Error(
				codes.PermissionDenied,
				"group owner must transfer ownership or dissolve the group",
			)

		default:
			l.Errorf("leave group: %v", err)
			return nil, status.Error(
				codes.Internal,
				"could not leave group",
			)
		}
	}

	return &social.LeaveGroupResp{
		MembershipVersion: int64(result.MembershipVersion),
	}, nil
}
