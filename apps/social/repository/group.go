package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/chanmineom/easy-chat/apps/social/model"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

const MaxGroupMembers = 500

var (
	ErrInvalidGroup             = errors.New("invalid group")
	ErrGroupNotFound            = errors.New("group not found")
	ErrGroupNotActive           = errors.New("group is not active")
	ErrGroupMemberNotFound      = errors.New("group member not found")
	ErrGroupMemberNotActive     = errors.New("group member is not active")
	ErrGroupOwnerCannotLeave    = errors.New("group owner cannot leave directly")
	ErrGroupPermissionDenied    = errors.New("group permission denied")
	ErrGroupTooLarge            = errors.New("group member limit exceeded")
	ErrGroupOwnerCannotBeKicked = errors.New("group owner cannot be kicked")
)

type CreateGroupInput struct {
	CreatorID    string
	Name         string
	AvatarURL    string
	Notification string
	MemberIDs    []string
}

type CreatedGroup struct {
	GroupID string
}

func (s *MySQLStore) CreateGroup(ctx context.Context, input CreateGroupInput) (CreatedGroup, error) {
	creatorID := strings.TrimSpace(input.CreatorID)
	name := strings.TrimSpace(input.Name)
	if creatorID == "" {
		return CreatedGroup{}, fmt.Errorf(
			"%w: creator id is required",
			ErrInvalidGroup,
		)
	}
	if name == "" || len([]rune(name)) > 128 {
		return CreatedGroup{}, fmt.Errorf(
			"%w: name must contain 1-128 characters",
			ErrInvalidGroup,
		)
	}
	if len(input.AvatarURL) > 512 {
		return CreatedGroup{}, fmt.Errorf(
			"%w: avatar url is too long",
			ErrInvalidGroup,
		)
	}
	if len(input.Notification) > 512 {
		return CreatedGroup{}, fmt.Errorf(
			"%w: notification is too long",
			ErrInvalidGroup,
		)
	}

	memberIDs, err := normalizeMembers(creatorID, input.MemberIDs)
	if err != nil {
		return CreatedGroup{}, err
	}

	groupID, err := newGroupID()
	if err != nil {
		return CreatedGroup{}, fmt.Errorf("generate group id: %w", err)
	}

	now := time.Now().UTC()
	avatarURL := strings.TrimSpace(input.AvatarURL)
	notification := strings.TrimSpace(input.Notification)

	err = s.conn.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		_, err := s.ExecCtx(
			ctx,
			`INSERT INTO chat_groups (
				id, name, avatar_url, notification,
				owner_id, status, membership_version,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, 1, 1, ?, ?)`,
			groupID, name, avatarURL, notification, creatorID, now, now,
		)
		if err != nil {
			return fmt.Errorf("insert group: %w", err)
		}

		memberSnapshots := make([]model.GroupMemberSnapshot, 0, len(memberIDs))
		for _, userID := range memberIDs {
			role := model.GroupRoleMember
			if userID == creatorID {
				role = model.GroupRoleOwner
			}

			_, err := s.ExecCtx(
				ctx,
				`INSERT INTO group_members (
					group_id, user_id, role, status,
					joined_at, left_at, created_at, updated_at
				) VALUES (?, ?, ?, 1, ?, NULL, ?, ?)`,
				groupID, userID, role, now, now, now,
			)
			if err != nil {
				return fmt.Errorf("insert group member %s: %w",
					userID,
					err,
				)
			}

			memberSnapshots = append(memberSnapshots, model.GroupMemberSnapshot{
				UserID:   userID,
				Role:     role,
				Status:   model.GroupMemberActive,
				JoinedAt: now,
			})
		}

		payload, err := json.Marshal(model.GroupCreatedEvent{
			GroupID:           groupID,
			Name:              name,
			AvatarURL:         avatarURL,
			Notification:      notification,
			OwnerID:           creatorID,
			MembershipVersion: 1,
			Members:           memberSnapshots,
			CreatedAt:         now,
		})
		if err != nil {
			return fmt.Errorf("encode group event: %w", err)
		}

		eventID := "evt_group_created_" + groupID

		_, err = s.ExecCtx(
			ctx,
			`INSERT INTO social_outbox (
				event_id, aggregate_type, aggregate_id,
				event_type, event_version, payload,
				status, attempts, available_at,
				created_at, updated_at) 
			VALUES (?, 'group', ?, 'group.created', 1, ?,
				1, 0, ?, ?, ?)`,
			eventID,
			groupID,
			string(payload),
			now,
			now,
			now,
		)
		if err != nil {
			return fmt.Errorf("insert social outbox: %w", err)
		}

		return nil
	})
	if err != nil {
		return CreatedGroup{}, err
	}

	return CreatedGroup{
		GroupID: groupID,
	}, nil
}

func normalizeMembers(creatorID string, memberIDs []string) ([]string, error) {
	unique := map[string]struct{}{
		creatorID: {},
	}

	for _, userID := range memberIDs {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			return nil, fmt.Errorf(
				"%w: member id is empty",
				ErrInvalidGroup,
			)
		}
		unique[userID] = struct{}{}
	}

	if len(unique) > MaxGroupMembers {
		return nil, fmt.Errorf(
			"%w: member count exceeds %d",
			ErrInvalidGroup,
			MaxGroupMembers,
		)
	}

	result := make([]string, 0, len(unique))
	for userID := range unique {
		result = append(result, userID)
	}
	sort.Strings(result)

	return result, nil
}

func newGroupID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}

	return "g_" + hex.EncodeToString(value[:]), nil
}

type LeaveGroupResult struct {
	MembershipVersion uint64
}

func (s *MySQLStore) LeaveGroup(ctx context.Context, groupID string, userID string) (LeaveGroupResult, error) {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	if groupID == "" || userID == "" {
		return LeaveGroupResult{}, fmt.Errorf(
			"%w: group_id and user_id are required",
			ErrInvalidGroup,
		)
	}

	var result LeaveGroupResult

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
				`SELECT
					owner_id,
					status,
					membership_version
				FROM
					chat_groups
				WHERE
					id = ? FOR UPDATE`,
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
			if group.OwnerID == userID {
				return ErrGroupOwnerCannotLeave
			}

			var member struct {
				Role   int `db:"role"`
				Status int `db:"status"`
			}

			err = s.QueryRowCtx(
				ctx,
				&member,
				`SELECT
					role,
					status
				FROM
					group_members
				WHERE
					group_id = ?
					AND user_id = ? FOR UPDATE`,
				groupID, userID,
			)
			if errors.Is(err, sqlx.ErrNotFound) {
				return ErrGroupMemberNotFound
			}
			if err != nil {
				return fmt.Errorf("lock group member: %w", err)
			}
			if member.Status != int(model.GroupMemberActive) {
				return ErrGroupMemberNotActive
			}

			now := time.Now().UTC()
			nextVersion := group.MembershipVersion + 1

			_, err = s.ExecCtx(
				ctx,
				`UPDATE group_members
				SET status = ?,
					left_at = ?,
					updated_at = ?
				WHERE
					group_id = ?
					AND user_id = ?`,
				model.GroupMemberLeft,
				now,
				now,
				groupID,
				userID,
			)
			if err != nil {
				return fmt.Errorf("update member left: %w", err)
			}

			_, err = s.ExecCtx(
				ctx,
				`UPDATE chat_groups
				SET membership_version = ?,
					updated_at = ?
				WHERE id = ?`,
				nextVersion,
				now,
				groupID,
			)
			if err != nil {
				return fmt.Errorf("increment group version: %w", err)
			}

			payload, err := json.Marshal(model.GroupMemberLeftEvent{
				GroupID:           groupID,
				UserID:            userID,
				ActorID:           userID,
				MembershipVersion: nextVersion,
				LeftAt:            now,
			})
			if err != nil {
				return fmt.Errorf("encode member left event: %w", err)
			}

			eventID := fmt.Sprintf(
				"evt_group_member_left_%s_%d",
				groupID,
				nextVersion,
			)

			_, err = s.ExecCtx(
				ctx,
				`INSERT INTO social_outbox (event_id, aggregate_type, aggregate_id, event_type, event_version, payload, STATUS, attempts, available_at, created_at, updated_at)
				VALUES (?, 'group', ?, 'group.member.left', ?, ?, 1, 0, ?, ?, ?)`,
				eventID,
				groupID,
				nextVersion,
				string(payload),
				now,
				now,
				now,
			)
			if err != nil {
				return fmt.Errorf("insert member left outbox: %w", err)
			}

			result.MembershipVersion = nextVersion
			return nil
		},
	)
	if err != nil {
		return LeaveGroupResult{}, err
	}

	return result, nil
}

type AddGroupMembersResult struct {
	MembershipVersion uint64
	AddedUserIDs      []string
}

func (s *MySQLStore) AddGroupMembers(ctx context.Context, groupID string, operatorID string, memberIDs []string) (AddGroupMembersResult, error) {
	groupID = strings.TrimSpace(groupID)
	operatorID = strings.TrimSpace(operatorID)

	if groupID == "" || operatorID == "" || len(memberIDs) == 0 {
		return AddGroupMembersResult{}, fmt.Errorf(
			"%w: group, operator and members are required",
			ErrInvalidGroup,
		)
	}

	targetIDs, err := normalizeMembers(operatorID, memberIDs)
	if err != nil {
		return AddGroupMembersResult{}, err
	}

	var result AddGroupMembersResult

	err = s.conn.TransactCtx(
		ctx,
		func(ctx context.Context, s sqlx.Session) error {
			var group struct {
				Status            int    `db:"status"`
				MembershipVersion uint64 `db:"membership_version"`
			}

			err := s.QueryRowCtx(
				ctx,
				&group,
				`SELECT status, membership_version
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

			var operator struct {
				Role   int `db:"role"`
				Status int `db:"status"`
			}

			err = s.QueryRowCtx(
				ctx,
				&operator,
				`SELECT role, status
				 FROM group_members
				 WHERE group_id = ? AND user_id = ?
				 FOR UPDATE`,
				groupID,
				operatorID,
			)
			if errors.Is(err, sqlx.ErrNotFound) {
				return ErrGroupPermissionDenied
			}
			if err != nil {
				return fmt.Errorf("load operator: %w", err)
			}
			if operator.Status != int(model.GroupMemberActive) {
				return ErrGroupPermissionDenied
			}
			if operator.Role != int(model.GroupRoleOwner) && operator.Role != int(model.GroupRoleAdmin) {
				return ErrGroupPermissionDenied
			}

			var activeCount int
			err = s.QueryRowCtx(
				ctx,
				&activeCount,
				`SELECT COUNT(*)
				 FROM group_members
				 WHERE group_id = ? AND status = ?`,
				groupID,
				model.GroupMemberActive,
			)
			if err != nil {
				return fmt.Errorf("count active members: %w", err)
			}

			now := time.Now().UTC()
			addedUserIDs := make([]string, 0, len(targetIDs))

			for _, userID := range targetIDs {
				var current struct {
					Status int `db:"status"`
				}

				err := s.QueryRowCtx(
					ctx,
					&current,
					`SELECT status
					 FROM group_members
					 WHERE group_id = ? AND user_id = ?
					 FOR UPDATE`,
					groupID,
					userID,
				)
				switch {
				case errors.Is(err, sqlx.ErrNotFound):
					if activeCount >= MaxGroupMembers {
						return ErrGroupTooLarge
					}

					_, err = s.ExecCtx(
						ctx,
						`INSERT INTO group_members (
							group_id, user_id, role, status,
							joined_at, left_at, created_at, updated_at
						) VALUES (?, ?, ?, 1, ?, NULL, ?, ?)`,
						groupID,
						userID,
						model.GroupRoleMember,
						now,
						now,
						now,
					)
					if err != nil {
						return fmt.Errorf(
							"insert group member %s: %w",
							userID,
							err,
						)
					}

					activeCount++
					addedUserIDs = append(addedUserIDs, userID)

				case err != nil:
					return fmt.Errorf(
						"load group member %s: %w",
						userID,
						err,
					)

				case current.Status == int(model.GroupMemberActive):
					continue

				default:
					if activeCount >= MaxGroupMembers {
						return ErrGroupTooLarge
					}

					_, err = s.ExecCtx(
						ctx,
						`UPDATE group_members
						 SET role = ?,
						     status = 1,
						     joined_at = ?,
						     left_at = NULL,
						     updated_at = ?
						 WHERE group_id = ? AND user_id = ?`,
						model.GroupRoleMember,
						now,
						now,
						groupID,
						userID,
					)
					if err != nil {
						return fmt.Errorf(
							"reactivate group member %s: %w",
							userID,
							err,
						)
					}

					activeCount++
					addedUserIDs = append(addedUserIDs, userID)
				}
			}

			if len(addedUserIDs) == 0 {
				result.MembershipVersion = group.MembershipVersion
				return nil
			}

			nextVersion := group.MembershipVersion + 1

			_, err = s.ExecCtx(
				ctx,
				`UPDATE chat_groups
				 SET membership_version = ?,
				     updated_at = ?
				 WHERE id = ?`,
				nextVersion,
				now,
				groupID,
			)
			if err != nil {
				return fmt.Errorf("increment group version: %w", err)
			}

			payload, err := json.Marshal(model.GroupMembersAddedEvent{
				GroupID:           groupID,
				ActorID:           operatorID,
				MembershipVersion: nextVersion,
				UserIDs:           addedUserIDs,
				JoinedAt:          now,
			})
			if err != nil {
				return fmt.Errorf("encode member added event: %w", err)
			}

			eventID := fmt.Sprintf(
				"evt_group_members_added_%s_%d",
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
				) VALUES (?, 'group', ?, 'group.member.added', ?, ?,
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
				return fmt.Errorf(
					"insert member added outbox: %w",
					err,
				)
			}

			result.MembershipVersion = nextVersion
			result.AddedUserIDs = addedUserIDs
			return nil
		},
	)
	if err != nil {
		return AddGroupMembersResult{}, err
	}

	return result, nil
}

type KickGroupMemberResult struct {
	MembershipVersion uint64
}

func (s *MySQLStore) KickGroupMember(ctx context.Context, groupID string, operatorID string, targetUserID string) (KickGroupMemberResult, error) {
	groupID = strings.TrimSpace(groupID)
	operatorID = strings.TrimSpace(operatorID)
	targetUserID = strings.TrimSpace(targetUserID)

	if groupID == "" || operatorID == "" || targetUserID == "" {
		return KickGroupMemberResult{}, fmt.Errorf(
			"%w: group, operator and target are required",
			ErrInvalidGroup,
		)
	}
	if operatorID == targetUserID {
		return KickGroupMemberResult{}, fmt.Errorf(
			"%w: use LeaveGroup to leave by yourself",
			ErrGroupPermissionDenied,
		)
	}

	var result KickGroupMemberResult

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
			if targetUserID == group.OwnerID {
				return ErrGroupOwnerCannotBeKicked
			}

			var operator struct {
				Role   int `db:"role"`
				Status int `db:"status"`
			}

			err = s.QueryRowCtx(
				ctx,
				&operator,
				`SELECT role, status
				 FROM group_members
				 WHERE group_id = ? AND user_id = ?
				 FOR UPDATE`,
				groupID,
				operatorID,
			)
			if errors.Is(err, sqlx.ErrNotFound) {
				return ErrGroupPermissionDenied
			}
			if err != nil {
				return fmt.Errorf("load operator: %w", err)
			}
			if operator.Status != int(model.GroupMemberActive) {
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
				targetUserID,
			)
			if errors.Is(err, sqlx.ErrNotFound) {
				return ErrGroupMemberNotFound
			}
			if err != nil {
				return fmt.Errorf("load target member: %w", err)
			}
			if target.Status != int(model.GroupMemberActive) {
				return ErrGroupMemberNotActive
			}

			switch model.GroupRole(operator.Role) {
			case model.GroupRoleOwner:
				// 群主可以移除管理员或普通成员。

			case model.GroupRoleAdmin:
				if target.Role != int(model.GroupRoleMember) {
					return ErrGroupPermissionDenied
				}

			default:
				return ErrGroupPermissionDenied
			}

			now := time.Now().UTC()
			nextVersion := group.MembershipVersion + 1

			_, err = s.ExecCtx(
				ctx,
				`UPDATE group_members
				 SET status = ?,
				     left_at = ?,
				     updated_at = ?
				 WHERE group_id = ? AND user_id = ?`,
				model.GroupMemberKicked,
				now,
				now,
				groupID,
				targetUserID,
			)
			if err != nil {
				return fmt.Errorf("update kicked member: %w", err)
			}

			_, err = s.ExecCtx(
				ctx,
				`UPDATE chat_groups
				 SET membership_version = ?,
				     updated_at = ?
				 WHERE id = ?`,
				nextVersion,
				now,
				groupID,
			)
			if err != nil {
				return fmt.Errorf("increment group version: %w", err)
			}

			payload, err := json.Marshal(model.GroupMemberKickedEvent{
				GroupID:           groupID,
				ActorID:           operatorID,
				UserID:            targetUserID,
				MembershipVersion: nextVersion,
				KickedAt:          now,
			})
			if err != nil {
				return fmt.Errorf("encode member kicked event: %w", err)
			}

			eventID := fmt.Sprintf(
				"evt_group_member_kicked_%s_%d",
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
				) VALUES (?, 'group', ?, 'group.member.kicked', ?, ?,
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
				return fmt.Errorf(
					"insert member kicked outbox: %w",
					err,
				)
			}

			result.MembershipVersion = nextVersion
			return nil
		},
	)
	if err != nil {
		return KickGroupMemberResult{}, err
	}

	return result, nil
}
