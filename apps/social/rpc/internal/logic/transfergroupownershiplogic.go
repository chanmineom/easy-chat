package logic

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type TransferGroupOwnershipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTransferGroupOwnershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TransferGroupOwnershipLogic {
	return &TransferGroupOwnershipLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TransferGroupOwnershipLogic) TransferGroupOwnership(in *social.TransferGroupOwnershipReq) (*social.TransferGroupOwnershipResp, error) {
	if in == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"request is required",
		)
	}

	result, err := l.svcCtx.Store.TransferGroupOwnership(
		l.ctx,
		in.GroupId,
		in.OperatorId,
		in.NewOwnerId,
	)
	if err != nil {
		return nil, mapGroupMutationError(l.Logger, err)
	}

	return &social.TransferGroupOwnershipResp{
		MembershipVersion: int64(result.MembershipVersion),
	}, nil
}
