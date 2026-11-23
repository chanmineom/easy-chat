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

type GetGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupLogic {
	return &GetGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetGroupLogic) GetGroup(in *social.GetGroupReq) (*social.GetGroupResp, error) {
	if in == nil || in.UserId == "" || in.GroupId == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"user and group are required",
		)
	}

	row, err := l.svcCtx.Store.GetGroupForUser(
		l.ctx,
		in.UserId,
		in.GroupId,
	)
	if err != nil {
		if errors.Is(err, repository.ErrGroupPermissionDenied) {
			return nil, status.Error(
				codes.PermissionDenied,
				"user is not an active group member",
			)
		}

		return nil, status.Error(
			codes.Internal,
			"could not load group",
		)
	}

	return &social.GetGroupResp{
		Group: &social.GroupSummary{
			GroupId:      row.ID,
			Name:         row.Name,
			AvatarUrl:    row.AvatarURL,
			Notification: row.Notification,
			OwnerId:      row.OwnerID,
			Role:         int32(row.Role),
			Status:       int32(row.Status),
			CreatedAt:    row.CreatedAt.UnixMilli(),
			UpdatedAt:    row.UpdatedAt.UnixMilli(),
		},
	}, nil
}
