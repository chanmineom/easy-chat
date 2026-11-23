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

type ChangeGroupMemberRoleResult struct {
	MembershipVersion uint64
}

func (s *MySQLStore) ChangeGroupMemberRole(
	ctx context.Context,
	groupID string,
	operatorID string,
	targetUserID string,
	newRole model.GroupRole,
) (ChangeGroupMemberRoleResult, error) {
	groupID = strings.TrimSpace(groupID)
	operatorID = strings.TrimSpace(operatorID)
	targetUserID = strings.TrimSpace(targetUserID)

	if groupID == "" || operatorID == "" || targetUserID == "" {
		return ChangeGroupMemberRoleResult{}, fmt.Errorf(
			"%w: group, operator and target are required",
			ErrInvalidGroup,
		)
	}
	if newRole != model.GroupRoleAdmin && newRole != model.GroupRoleMember {
		return ChangeGroupMemberRoleResult{}, fmt.Errorf(
			"%w: role must be admin or member",
			ErrInvalidGroup,
		)
	}

	var result ChangeGroupMemberRoleResult

	err := s.conn.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
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
		if targetUserID == group.OwnerID {
			return fmt.Errorf(
				"%w: owner role cannot be changed here",
				ErrGroupPermissionDenied,
			)
		}

		var target struct {
			Role   int `db:"role"`
			Status int `db:"status"`
		}

		err = s.QueryRowCtx(
			ctx,
			&target,
			`SELECT role, status
			 FROM group_members
			 WHERE group_id = ? AND user_id = ?
			 FOR UPDATE`,
			groupID,
			targetUserID,
		)
		if errors.Is(err, sqlx.ErrNotFound) {
			return ErrGroupMemberNotActive
		}
		if err != nil {
			return fmt.Errorf("lock target member: %w", err)
		}
		if target.Status != int(model.GroupMemberActive) {
			return ErrGroupMemberNotActive
		}

		oldRole := model.GroupRole(target.Role)
		if oldRole == newRole {
			result.MembershipVersion = group.MembershipVersion
			return nil
		}

		now := time.Now().UTC()
		nextVersion := group.MembershipVersion + 1

		_, err = s.ExecCtx(
			ctx,
			`UPDATE group_members
			 SET role = ?, updated_at = ?
			 WHERE group_id = ? AND user_id = ?`,
			newRole,
			now,
			groupID,
			targetUserID,
		)
		if err != nil {
			return fmt.Errorf("update member role: %w", err)
		}

		_, err = s.ExecCtx(
			ctx,
			`UPDATE chat_groups
			 SET membership_version = ?, updated_at = ?
			 WHERE id = ?`,
			nextVersion,
			now,
			groupID,
		)
		if err != nil {
			return fmt.Errorf("increment group version: %w", err)
		}

		payload, err := json.Marshal(model.GroupRoleChangedEvent{
			GroupID:           groupID,
			ActorID:           operatorID,
			UserID:            targetUserID,
			OldRole:           oldRole,
			NewRole:           newRole,
			MembershipVersion: nextVersion,
			ChangedAt:         now,
		})
		if err != nil {
			return fmt.Errorf("encode role event: %w", err)
		}

		eventID := fmt.Sprintf(
			"evt_group_role_changed_%s_%d",
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
			) VALUES (?, 'group', ?, 'group.role.changed', ?, ?,
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
			return fmt.Errorf("insert role outbox: %w", err)
		}

		result.MembershipVersion = nextVersion
		return nil
	})
	if err != nil {
		return ChangeGroupMemberRoleResult{}, err
	}

	return result, nil
}

type TransferGroupOwnershipResult struct {
	MembershipVersion uint64
}

func (s *MySQLStore) TransferGroupOwnership(
	ctx context.Context,
	groupID string,
	operatorID string,
	newOwnerID string,
) (TransferGroupOwnershipResult, error) {
	groupID = strings.TrimSpace(groupID)
	operatorID = strings.TrimSpace(operatorID)
	newOwnerID = strings.TrimSpace(newOwnerID)

	if groupID == "" || operatorID == "" || newOwnerID == "" {
		return TransferGroupOwnershipResult{}, fmt.Errorf(
			"%w: group, operator and new owner are required",
			ErrInvalidGroup,
		)
	}
	if operatorID == newOwnerID {
		return TransferGroupOwnershipResult{}, fmt.Errorf(
			"%w: new owner must be different from current owner",
			ErrInvalidGroup,
		)
	}

	var result TransferGroupOwnershipResult

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

			var target struct {
				Role   int `db:"role"`
				Status int `db:"status"`
			}

			err = s.QueryRowCtx(
				ctx,
				&target,
				`SELECT role, status
				 FROM group_members
				 WHERE group_id = ? AND user_id = ?
				 FOR UPDATE`,
				groupID,
				newOwnerID,
			)
			if errors.Is(err, sqlx.ErrNotFound) {
				return ErrGroupMemberNotFound
			}
			if err != nil {
				return fmt.Errorf("lock new owner: %w", err)
			}
			if target.Status != int(model.GroupMemberActive) {
				return ErrGroupMemberNotFound
			}

			now := time.Now().UTC()
			nextVersion := group.MembershipVersion + 1

			_, err = s.ExecCtx(
				ctx,
				`UPDATE group_members
				 SET role = ?
				 WHERE group_id = ? AND user_id = ?`,
				model.GroupRoleOwner,
				groupID,
				newOwnerID,
			)
			if err != nil {
				return fmt.Errorf("promote new owner: %w", err)
			}

			_, err = s.ExecCtx(
				ctx,
				`UPDATE group_members
				 SET role = ?
				 WHERE group_id = ? AND user_id = ?`,
				model.GroupRoleMember,
				groupID,
				operatorID,
			)
			if err != nil {
				return fmt.Errorf("demote previous owner: %w", err)
			}

			_, err = s.ExecCtx(
				ctx,
				`UPDATE chat_groups
				 SET owner_id = ?,
				 	membership_version = ?,
				 	updated_at = ?
				 WHERE id = ?`,
				newOwnerID,
				nextVersion,
				now,
				groupID,
			)
			if err != nil {
				return fmt.Errorf("update group owner: %w", err)
			}

			payload, err := json.Marshal(model.GroupOwnerTransferredEvent{
				GroupID:           groupID,
				PreviousOwnerID:   operatorID,
				NewOwnerID:        newOwnerID,
				MembershipVersion: nextVersion,
				TransferredAt:     now,
			})
			if err != nil {
				return fmt.Errorf("encode owner transfer event: %w", err)
			}

			eventID := fmt.Sprintf(
				"evt_group_owner_transferred_%s_%d",
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
				) VALUES (?, 'group', ?, 'group.owner.transferred', ?, ?,
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
				return fmt.Errorf("insert owner outbox: %w", err)
			}

			result.MembershipVersion = nextVersion
			return nil
		},
	)
	if err != nil {
		return TransferGroupOwnershipResult{}, err
	}

	return result, nil
}
