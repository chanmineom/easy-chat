package logic

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type DissolveGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDissolveGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DissolveGroupLogic {
	return &DissolveGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DissolveGroupLogic) DissolveGroup(in *social.DissolveGroupReq) (*social.DissolveGroupResp, error) {
	if in == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"request is required",
		)
	}

	result, err := l.svcCtx.Store.DissolveGroup(
		l.ctx,
		in.GroupId,
		in.OperatorId,
	)
	if err != nil {
		return nil, mapGroupMutationError(l.Logger, err)
	}

	return &social.DissolveGroupResp{
		MembershipVersion: int64(result.MembershipVersion),
	}, nil
}
