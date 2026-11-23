// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package group

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/social/api/internal/auth"
	"github.com/chanmineom/easy-chat/apps/social/api/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/api/internal/types"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"

	"github.com/zeromicro/go-zero/core/logx"
)

type TransferGroupOwnershipLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// transfer group ownership
func NewTransferGroupOwnershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TransferGroupOwnershipLogic {
	return &TransferGroupOwnershipLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TransferGroupOwnershipLogic) TransferGroupOwnership(req *types.TransferGroupOwnershipReq) (resp *types.TransferGroupOwnershipResp, err error) {
	userID, err := auth.UserID(l.ctx)
	if err != nil {
		return nil, err
	}

	response, err := l.svcCtx.Social.TransferGroupOwnership(
		l.ctx,
		&social.TransferGroupOwnershipReq{
			GroupId:    req.GroupId,
			OperatorId: userID,
			NewOwnerId: req.NewOwnerId,
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.TransferGroupOwnershipResp{
		MembershipVersion: response.MembershipVersion,
	}, nil
}
