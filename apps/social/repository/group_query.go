package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type GroupSummaryRecord struct {
	ID           string    `db:"id"`
	Name         string    `db:"name"`
	AvatarURL    string    `db:"avatar_url"`
	Notification string    `db:"notification"`
	OwnerID      string    `db:"owner_id"`
	Role         int       `db:"role"`
	Status       int       `db:"status"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type GroupMemberRecord struct {
	UserID   string    `db:"user_id"`
	Role     int       `db:"role"`
	Status   int       `db:"status"`
	JoinedAt time.Time `db:"joined_at"`
}

func (s *MySQLStore) ListUserGroups(ctx context.Context, userID string) ([]GroupSummaryRecord, error) {
	var result []GroupSummaryRecord

	err := s.conn.QueryRowsCtx(
		ctx,
		&result,
		`SELECT g.id, g.name, g.avatar_url, g.notification, g.owner_id, gm.role, g.status, g.created_at, g.updated_at
		 FROM group_members gm
		 INNER JOIN chat_groups g ON gm.group_id = g.id
		 WHERE gm.user_id = ?
		 	AND gm.status = 1
		 	AND g.status = 1
		 ORDER BY g.updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list user groups: %w", err)
	}

	return result, nil
}

func (s *MySQLStore) GetGroupForUser(
	ctx context.Context,
	userID string,
	groupID string,
) (GroupSummaryRecord, error) {
	var result GroupSummaryRecord

	err := s.conn.QueryRowCtx(
		ctx,
		&result,
		`SELECT g.id, g.name, g.avatar_url, g.notification, g.owner_id, gm.role, g.status, g.created_at, g.updated_at
		 FROM group_members gm
		 INNER JOIN chat_groups g ON gm.group_id = g.id
		 WHERE gm.user_id = ?
		 	AND gm.status = 1
		 	AND gm.group_id = ?
		 LIMIT 1`,
		userID,
		groupID,
	)
	if errors.Is(err, sqlx.ErrNotFound) {
		return GroupSummaryRecord{}, ErrGroupPermissionDenied
	}
	if err != nil {
		return GroupSummaryRecord{}, fmt.Errorf(
			"get group for user: %w",
			err,
		)
	}

	return result, nil
}

func (s *MySQLStore) ListActiveGroupMembers(
	ctx context.Context,
	requesterID string,
	groupID string,
) ([]GroupMemberRecord, error) {
	if _, err := s.GetGroupForUser(
		ctx,
		requesterID,
		groupID,
	); err != nil {
		return nil, err
	}

	var result []GroupMemberRecord

	err := s.conn.QueryRowsCtx(
		ctx,
		&result,
		`SELECT user_id, role, status, joined_at
		 FROM group_members
		 WHERE group_id = ?
		 	AND status = 1
		 ORDER BY role ASC, joined_at ASC`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("list group members: %w", err)
	}

	return result, nil
}
