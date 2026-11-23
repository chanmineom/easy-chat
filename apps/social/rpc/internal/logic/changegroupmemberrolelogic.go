package logic

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/social/model"
	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChangeGroupMemberRoleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChangeGroupMemberRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangeGroupMemberRoleLogic {
	return &ChangeGroupMemberRoleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ChangeGroupMemberRoleLogic) ChangeGroupMemberRole(in *social.ChangeGroupMemberRoleReq) (*social.ChangeGroupMemberRoleResp, error) {
	if in == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"request is required",
		)
	}

	result, err := l.svcCtx.Store.ChangeGroupMemberRole(
		l.ctx,
		in.GroupId,
		in.OperatorId,
		in.UserId,
		model.GroupRole(in.Role),
	)
	if err != nil {
		return nil, mapGroupMutationError(l.Logger, err)
	}

	return &social.ChangeGroupMemberRoleResp{
		MembershipVersion: int64(result.MembershipVersion),
	}, nil
}
