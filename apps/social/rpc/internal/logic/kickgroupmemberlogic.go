package logic

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type KickGroupMemberLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewKickGroupMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KickGroupMemberLogic {
	return &KickGroupMemberLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *KickGroupMemberLogic) KickGroupMember(in *social.KickGroupMemberReq) (*social.KickGroupMemberResp, error) {
	if in == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"request is required",
		)
	}

	result, err := l.svcCtx.Store.KickGroupMember(l.ctx, in.GroupId, in.OperatorId, in.UserId)
	if err != nil {
		return nil, mapGroupMutationError(l.Logger, err)
	}

	return &social.KickGroupMemberResp{
		MembershipVersion: int64(result.MembershipVersion),
	}, nil
}
