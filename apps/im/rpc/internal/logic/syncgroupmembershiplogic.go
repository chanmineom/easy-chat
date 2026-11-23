package logic

import (
	"context"
	"errors"

	"github.com/chanmineom/easy-chat/apps/im/repository"
	"github.com/chanmineom/easy-chat/apps/im/rpc/im"
	"github.com/chanmineom/easy-chat/apps/im/rpc/internal/svc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type SyncGroupMembershipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSyncGroupMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SyncGroupMembershipLogic {
	return &SyncGroupMembershipLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SyncGroupMembershipLogic) SyncGroupMembership(in *im.SyncGroupMembershipReq) (*im.SyncGroupMembershipResp, error) {
	if in == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"request is required",
		)
	}
	if in.GroupId == "" || in.Version <= 0 {
		return nil, status.Error(
			codes.InvalidArgument,
			"group_id and positive version are required",
		)
	}

	switch in.Action {
	case "left":
		if len(in.MemberIds) != 1 {
			return nil, status.Error(
				codes.InvalidArgument,
				"left event requires exactly one member",
			)
		}

		err := l.svcCtx.Mongo.ApplyGroupMemberLeft(
			l.ctx,
			in.GroupId,
			in.MemberIds[0],
			uint64(in.Version),
		)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrConversationNotFound),
				errors.Is(err, repository.ErrMembershipMemberNotFound):
				l.Errorf(
					"membership projection not found group=%s: %v",
					in.GroupId,
					err,
				)
				return &im.SyncGroupMembershipResp{}, nil
			default:
				l.Errorf("apply group membership: %v", err)
				return nil, status.Error(
					codes.Internal,
					"could not apply group membership",
				)
			}
		}

	case "added":
		err := l.svcCtx.Mongo.ApplyGroupMembersAdded(
			l.ctx,
			in.GroupId,
			in.MemberIds,
			uint64(in.Version),
		)
		if err != nil {
			return nil, status.Error(
				codes.Internal,
				"could not add group members",
			)
		}

	case "kicked":
		if len(in.MemberIds) != 1 {
			return nil, status.Error(
				codes.InvalidArgument,
				"kicked event requires exactly one member",
			)
		}

		err := l.svcCtx.Mongo.ApplyGroupMemberLeft(l.ctx, in.GroupId, in.MemberIds[0], uint64(in.Version))
		if err != nil {
			return nil, status.Error(
				codes.Internal,
				"could not remove group member",
			)
		}

	case "role.changed":
		if len(in.MemberIds) != 1 {
			return nil, status.Error(
				codes.InvalidArgument,
				"role change requires exactly one member",
			)
		}

		err := l.svcCtx.Mongo.ApplyGroupRoleChanged(
			l.ctx,
			in.GroupId,
			in.MemberIds[0],
			in.GroupRole,
			uint64(in.Version),
		)
		if err != nil {
			l.Errorf("apply group role changed: %v", err)
			return nil, status.Error(
				codes.Internal,
				"could not apply group role",
			)
		}

	case "owner.transferred":
		if len(in.MemberIds) != 1 {
			return nil, status.Error(
				codes.InvalidArgument,
				"owner transfer requires one new owner",
			)
		}

		err := l.svcCtx.Mongo.ApplyGroupOwnerTransferred(
			l.ctx,
			in.GroupId,
			in.ActorId,
			in.MemberIds[0],
			uint64(in.Version),
		)
		if err != nil {
			l.Errorf("apply group owner transfer: %v", err)
			return nil, status.Error(
				codes.Internal,
				"could not apply owner transfer",
			)
		}

	case "dissolved":
		err := l.svcCtx.Mongo.ApplyGroupDissolved(
			l.ctx,
			in.GroupId,
			uint64(in.Version),
		)
		if err != nil {
			l.Errorf("apply group dissolved: %v", err)
			return nil, status.Error(
				codes.Internal,
				"could not apply group dissolution",
			)
		}

	default:
		return nil, status.Error(
			codes.Unimplemented,
			"membership action is not implemented",
		)
	}

	return &im.SyncGroupMembershipResp{}, nil
}
