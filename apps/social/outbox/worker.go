package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/rpc/imclient"
	"github.com/chanmineom/easy-chat/apps/social/model"
	"github.com/chanmineom/easy-chat/apps/social/repository"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Worker struct {
	store *repository.MySQLStore
	imRpc imclient.Im

	pollInterval time.Duration
	lease        time.Duration
	baseBackoff  time.Duration
	maxBackoff   time.Duration
	maxAttempts  int
}

func NewWorker(store *repository.MySQLStore, imRpc imclient.Im) *Worker {
	return &Worker{
		store:        store,
		imRpc:        imRpc,
		pollInterval: 500 * time.Millisecond,
		lease:        30 * time.Second,
		baseBackoff:  time.Second,
		maxBackoff:   time.Minute,
		maxAttempts:  20,
	}
}

func (w *Worker) SyncNext(ctx context.Context) (bool, error) {
	now := time.Now().UTC()

	event, found, err := w.store.ClaimNextOutboxEvent(
		ctx,
		now,
		w.lease,
	)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}

	syncErr := w.sync(ctx, event)
	if syncErr == nil {
		if err := w.store.MarkOutboxSent(
			ctx,
			event.EventID,
			time.Now().UTC(),
		); err != nil {
			return true, err
		}
		return true, nil
	}

	permanent := w.permanent(syncErr) || event.Attempts >= w.maxAttempts
	retryAt := time.Now().UTC().Add(w.retryDelay(event.Attempts))

	if err := w.store.MarkOutboxFailed(
		ctx,
		event.EventID,
		retryAt,
		syncErr,
		permanent,
	); err != nil {
		return true, fmt.Errorf(
			"sync failed: %v; mark failed: %w",
			syncErr,
			err,
		)
	}

	return true, fmt.Errorf(
		"sync social outbox event %s: %w",
		event.EventID,
		syncErr,
	)
}

func (w *Worker) sync(ctx context.Context, event repository.SocialOutboxEvent) error {
	switch event.EventType {
	case "group.created":
		var payload model.GroupCreatedEvent
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode group.created payload: %w", err)
		}
		if payload.GroupID == "" || payload.OwnerID == "" {
			return fmt.Errorf("group.created payload is incomplete")
		}

		memberIDs := make([]string, 0, len(payload.Members))
		for _, member := range payload.Members {
			if member.UserID != "" {
				memberIDs = append(memberIDs, member.UserID)
			}
		}

		_, err := w.imRpc.CreateGroupConversation(ctx,
			&imclient.CreateGroupConversationReq{
				GroupId:   payload.GroupID,
				CreatorId: payload.OwnerID,
				MemberIds: memberIDs,
				MembershipVersion: int64(payload.MembershipVersion),
			},
		)
		if err != nil {
			return fmt.Errorf(
				"create IM group conversation: %w",
				err,
			)
		}

		return nil

	case "group.member.left":
		var payload model.GroupMemberLeftEvent
		if err := json.Unmarshal(
			event.Payload,
			&payload,
		); err != nil {
			return fmt.Errorf(
				"decode group.member.left payload: %w",
				err,
			)
		}

		_, err := w.imRpc.SyncGroupMembership(
			ctx,
			&imclient.SyncGroupMembershipReq{
				GroupId: payload.GroupID,
				Version: int64(payload.MembershipVersion),
				Action:  "left",
				ActorId: payload.ActorID,
				MemberIds: []string{
					payload.UserID,
				},
			},
		)
		if err != nil {
			return fmt.Errorf(
				"sync group member left: %w",
				err,
			)
		}
		return nil

	case "group.member.added":
		var payload model.GroupMembersAddedEvent
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}

		_, err := w.imRpc.SyncGroupMembership(
			ctx,
			&imclient.SyncGroupMembershipReq{
				GroupId:   payload.GroupID,
				Version:   int64(payload.MembershipVersion),
				Action:    "added",
				ActorId:   payload.ActorID,
				MemberIds: payload.UserIDs,
			},
		)
		return err

	case "group.role.changed":
		var payload model.GroupRoleChangedEvent
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}

		_, err := w.imRpc.SyncGroupMembership(
			ctx,
			&imclient.SyncGroupMembershipReq{
				GroupId: payload.GroupID,
				Version: int64(payload.MembershipVersion),
				Action: "role.changed",
				ActorId: payload.ActorID,
				MemberIds: []string{payload.UserID},
				GroupRole: int32(payload.NewRole),
			},
		)
		return err

	case "group.owner.transferred":
		var payload model.GroupOwnerTransferredEvent
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}

		_, err := w.imRpc.SyncGroupMembership(
			ctx,
			&imclient.SyncGroupMembershipReq{
				GroupId: payload.GroupID,
				Version: int64(payload.MembershipVersion),
				Action:  "owner.transferred",
				ActorId: payload.PreviousOwnerID,
				MemberIds: []string{payload.NewOwnerID},
				GroupRole: int32(model.GroupRoleOwner),
			},
		)
		return err

	case "group.dissolved":
		var payload model.GroupDissolveEvent
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf(
				"decode group.dissolved payload: %w",
				err,
			)
		}

		_, err := w.imRpc.SyncGroupMembership(
			ctx,
			&imclient.SyncGroupMembershipReq{
				GroupId: payload.GroupID,
				Version: int64(payload.MembershipVersion),
				Action: "dissolved",
				ActorId: payload.ActorID,
			},
		)
		return err

	case "friend.created":
		var payload model.FriendshipCreatedEvent
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf(
				"decode friend.created payload: %w",
				err,
			)
		}

		_, err := w.imRpc.SetUpUserConversation(
			ctx,
			&imclient.SetUpUserConversationReq{
				SendId: payload.UserA,
				RecvId: payload.UserB,
				ChatType: 2,
			},
		)
		if err != nil {
			return fmt.Errorf(
				"create direct conversation: %w",
				err,
			)
		}

		return nil

	default:
		return fmt.Errorf(
			"unsupported social outbox event type %q",
			event.EventType,
		)
	}
}

func (w *Worker) permanent(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch st.Code() {
	case codes.InvalidArgument, codes.AlreadyExists, codes.PermissionDenied:
		return true
	default:
		return false
	}
}

func (w *Worker) retryDelay(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}

	delay := w.baseBackoff
	for i := 1; i < attempts && delay < w.maxBackoff; i++ {
		delay *= 2
	}
	if delay > w.maxBackoff {
		delay = w.maxBackoff
	}

	return delay
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		processed, err := w.SyncNext(ctx)
		if err != nil {
			logx.WithContext(ctx).Errorf(
				"social outbox worker: %v",
				err,
			)
		}
		if processed {
			continue
		}

		timer := time.NewTimer(w.pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
