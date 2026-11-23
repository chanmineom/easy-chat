package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chanmineom/easy-chat/apps/social/model"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type DissolveGroupResult struct {
	MembershipVersion uint64
}

func (s *MySQLStore) DissolveGroup(
	ctx context.Context,
	groupID string,
	operatorID string,
) (DissolveGroupResult, error) {
	groupID = strings.TrimSpace(groupID)
	operatorID = strings.TrimSpace(operatorID)
	if groupID == "" || operatorID == "" {
		return DissolveGroupResult{}, fmt.Errorf(
			"%w: group and operator are required",
			ErrInvalidGroup,
		)
	}

	var result DissolveGroupResult

	err := s.conn.TransactCtx(
		ctx,
		func(ctx context.Context, s sqlx.Session) error {
			var group struct {
				OwnerID           string `db:"owner_id"`
				Status            int    `db:"status"`
				MembershipVersion uint64 `db:"membership_version"`
			}

			err := s.QueryRowCtx(
				ctx,
				&group,
				`SELECT owner_id, status, membership_version
				 FROM chat_groups
				 WHERE id = ?
				 FOR UPDATE`,
				groupID,
			)
			if errors.Is(err, sqlx.ErrNotFound) {
				return ErrGroupNotFound
			}
			if err != nil {
				return fmt.Errorf("lock group: %w", err)
			}
			if group.Status != 1 {
				return ErrGroupNotActive
			}
			if group.OwnerID != operatorID {
				return ErrGroupPermissionDenied
			}

			now := time.Now().UTC()
			nextVersion := group.MembershipVersion + 1

			_, err = s.ExecCtx(
				ctx,
				`UPDATE chat_groups
				 SET status = 3,
				 	membership_version = ?,
				 	updated_at = ?
				 WHERE id = ?`,
				nextVersion,
				now,
				groupID,
			)
			if err != nil {
				return fmt.Errorf("dissolve group: %w", err)
			}

			_, err = s.ExecCtx(
				ctx,
				`UPDATE group_members
				 SET status = ?,
				 	left_at = ?,
				 	updated_at = ?
				 WHERE group_id = ? AND status = ?`,
				model.GroupMemberDissolved,
				now,
				now,
				groupID,
				model.GroupMemberActive,
			)
			if err != nil {
				return fmt.Errorf("close group members: %w", err)
			}

			payload, err := json.Marshal(model.GroupDissolveEvent{
				GroupID:           groupID,
				ActorID:             operatorID,
				MembershipVersion: nextVersion,
				DissolvedAt:       now,
			})
			if err != nil {
				return fmt.Errorf("encode dissolve event: %w", err)
			}

			eventID := fmt.Sprintf(
				"evt_group_dissolved_%s_%d",
				groupID,
				nextVersion,
			)

			_, err = s.ExecCtx(
				ctx,
				`INSERT INTO social_outbox (
					event_id, aggregate_type, aggregate_id,
					event_type, event_version, payload,
					status, attempts, available_at,
					created_at, updated_at
				) VALUES (?, 'group', ?, 'group.dissolved', ?, ?,
					1, 0, ?, ?, ?)`,
				eventID,
				groupID,
				nextVersion,
				string(payload),
				now,
				now,
				now,
			)
			if err != nil {
				return fmt.Errorf("insert dissolve outbox: %w", err)
			}

			result.MembershipVersion = nextVersion
			return nil
		},
	)
	if err != nil {
		return DissolveGroupResult{}, err
	}

	return result, nil
}
