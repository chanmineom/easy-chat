package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chanmineom/easy-chat/apps/social/model"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func normalizedPair(a, b string) (string, string, error) {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)

	if a == "" || b == "" {
		return "", "", fmt.Errorf(
			"%w: both user ids are required",
			ErrInvalidGroup,
		)
	}
	if a == b {
		return "", "", ErrCannotFriendSelf
	}
	if a > b {
		a, b = b, a
	}

	return a, b, nil
}

type SendFriendRequestResult struct {
	RequestID int64
}

func (s *MySQLStore) SendFriendRequest(
	ctx context.Context,
	requesterID string,
	recipientID string,
	message string,
) (SendFriendRequestResult, error) {
	requesterID = strings.TrimSpace(requesterID)
	recipientID = strings.TrimSpace(recipientID)
	message = strings.TrimSpace(message)

	userA, userB, err := normalizedPair(requesterID, recipientID)
	if err != nil {
		return SendFriendRequestResult{}, err
	}
	if len(message) > 255 {
		return SendFriendRequestResult{}, fmt.Errorf(
			"friend request message is too long",
		)
	}

	var requestID int64

	err = s.conn.TransactCtx(
		ctx,
		func(ctx context.Context, s sqlx.Session) error {
			var existing int64
			err := s.QueryRowCtx(
				ctx,
				&existing,
				`SELECT id
				 FROM friendships
				 WHERE user_a = ? AND user_b = ? AND status = 1
				 LIMIT 1`,
				userA,
				userB,
			)
			if err == nil {
				return ErrAlreadyFriends
			}
			if !errors.Is(err, sqlx.ErrNotFound) {
				return fmt.Errorf("check friendship: %w", err)
			}

			now := time.Now().UTC()
			result, err := s.ExecCtx(
				ctx,
				`INSERT INTO friend_requests (
				 	requester_id, recipient_id, message,
				 	status, created_at, updated_at
				 ) VALUES (?, ?, ?, 1, ?, ?)`,
				requesterID,
				recipientID,
				message,
				now,
				now,
			)
			if err != nil {
				var mysqlErr *mysqldriver.MySQLError
				if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
					return ErrFriendRequestExists
				}
				return fmt.Errorf("insert friend request: %w", err)
			}

			requestID, err = result.LastInsertId()
			if err != nil {
				return fmt.Errorf("get request id: %w", err)
			}

			return nil
		},
	)
	if err != nil {
		return SendFriendRequestResult{}, err
	}

	return SendFriendRequestResult{
		RequestID: requestID,
	}, nil
}

type HandleFriendRequestResult struct {
	FriendshipID int64
}

func (s *MySQLStore) HandleFriendRequest(
	ctx context.Context,
	requestID int64,
	handlerID string,
	accept bool,
) (HandleFriendRequestResult, error) {
	handlerID = strings.TrimSpace(handlerID)
	if requestID <= 0 || handlerID == "" {
		return HandleFriendRequestResult{}, fmt.Errorf(
			"request id and handler id are required",
		)
	}

	var result HandleFriendRequestResult

	err := s.conn.TransactCtx(
		ctx,
		func(ctx context.Context, s sqlx.Session) error {
			var request struct {
				RequesterID string `db:"requester_id"`
				RecipientID string `db:"recipient_id"`
				Status      int    `db:"status"`
			}

			err := s.QueryRowCtx(
				ctx,
				&request,
				`SELECT requester_id, recipient_id, status
				 FROM friend_requests
				 WHERE id = ?
				 FOR UPDATE`,
				requestID,
			)
			if errors.Is(err, sqlx.ErrNotFound) {
				return ErrFriendRequestNotFound
			}
			if err != nil {
				return fmt.Errorf("lock friend request: %w", err)
			}
			if request.RecipientID != handlerID {
				return ErrFriendRequestForbidden
			}

			now := time.Now().UTC()

			if !accept {
				_, err = s.ExecCtx(
					ctx,
					`UPDATE friend_requests
					 SET status = 3,
					 	handled_at = ?,
					 	updated_at = ?
					 WHERE id = ?`,
					now,
					now,
					requestID,
				)
				return err
			}

			userA, userB, err := normalizedPair(
				request.RequesterID,
				request.RecipientID,
			)
			if err != nil {
				return err
			}

			var friendship struct {
				ID      int64  `db:"id"`
				Status  int    `db:"status"`
				Version uint64 `db:"version"`
			}

			err = s.QueryRowCtx(
				ctx,
				&friendship,
				`SELECT id, status, version
				 FROM friendships
				 WHERE user_a = ? AND user_b = ?
				 FOR UPDATE`,
				userA,
				userB,
			)

			switch {
			case errors.Is(err, sqlx.ErrNotFound):
				insert, err := s.ExecCtx(
					ctx,
					`INSERT INTO friendships (
						user_a, user_b, status, version,
						created_at, updated_at
					) VALUES (?, ?, 1, 1, ?, ?)`,
					userA,
					userB,
					now,
					now,
				)
				if err != nil {
					return fmt.Errorf("insert friendship: %w", err)
				}

				friendship.ID, err = insert.LastInsertId()
				if err != nil {
					return err
				}
				friendship.Status = 1
				friendship.Version = 1

			case err != nil:
				return fmt.Errorf("lock friendship: %w", err)

			case friendship.Status == 1:
				return ErrAlreadyFriends

			default:
				newVersion := friendship.Version + 1
				_, err = s.ExecCtx(
					ctx,
					`UPDATE friendships
					 SET status = 1,
					 	version = ?,
					 	updated_at = ?
					 WHERE id = ?`,
					newVersion,
					now,
					friendship.ID,
				)
				if err != nil {
					return fmt.Errorf("reactivate friendship: %w", err)
				}

				friendship.Status = 1
				friendship.Version = newVersion
			}

			_, err = s.ExecCtx(
				ctx,
				`UPDATE friend_requests
				 SET status = 2,
				 	handled_at = ?,
				 	updated_at = ?
				 WHERE id = ?`,
				now,
				now,
				requestID,
			)
			if err != nil {
				return fmt.Errorf("accept friend request: %w", err)
			}

			payload, err := json.Marshal(model.FriendshipCreatedEvent{
				FriendshipID: friendship.ID,
				UserA:        userA,
				UserB:        userB,
				Version:      friendship.Version,
				CreatedAt:    now,
			})
			if err != nil {
				return fmt.Errorf("encode friendship event: %w", err)
			}

			eventID := fmt.Sprintf(
				"evt_friendship_created_%d_%d",
				friendship.ID,
				friendship.Version,
			)

			_, err = s.ExecCtx(
				ctx,
				`INSERT INTO social_outbox (
					event_id, aggregate_type, aggregate_id,
					event_type, event_version, payload,
					status, attempts, available_at,
					created_at, updated_at
				) VALUES (?, 'friendship', ?, 'friend.created', ?, ?,
					1, 0, ?, ?, ?)`,
				eventID,
				strconv.FormatInt(friendship.ID, 10),
				friendship.Version,
				string(payload),
				now,
				now,
				now,
			)
			if err != nil {
				return fmt.Errorf("insert friendship outbox: %w", err)
			}

			result.FriendshipID = friendship.ID
			return nil
		},
	)
	if err != nil {
		return HandleFriendRequestResult{}, err
	}

	return result, nil
}

func (s *MySQLStore) IsFriend(ctx context.Context, userA, userB string) (bool, error) {
	userA, userB, err := normalizedPair(userA, userB)
	if err != nil {
		if errors.Is(err, ErrCannotFriendSelf) {
			return false, nil
		}
		return false, err
	}

	var count int
	err = s.conn.QueryRowCtx(
		ctx,
		&count,
		`SELECT COUNT(*)
		 FROM friendships
		 WHERE user_a = ? AND user_b = ? AND status = 1`,
		userA,
		userB,
	)
	if err != nil {
		return false, fmt.Errorf("check friendship: %w", err)
	}

	return count > 0, nil
}

func (s *MySQLStore) ListFriendIDs(ctx context.Context, userID string) ([]string, error) {
	var rows []struct {
		UserA string `db:"user_a"`
		UserB string `db:"user_b"`
	}

	err := s.conn.QueryRowsCtx(
		ctx,
		&rows,
		`SELECT user_a, user_b
		 FROM friendships
		 WHERE status = 1
		 	AND (user_a = ? OR user_b = ?)
		 ORDER BY updated_at DESC`,
		userID,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list friend ids: %w", err)
	}

	result := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.UserA == userID {
			result = append(result, row.UserB)
		} else {
			result = append(result, row.UserA)
		}
	}

	return result, nil
}

type RemoveFriendResult struct {
	FriendshipID int64
	Version      uint64
}

func (s *MySQLStore) RemoveFriend(
	ctx context.Context,
	userID string,
	friendID string,
) (RemoveFriendResult, error) {
	userA, userB, err := normalizedPair(userID, friendID)
	if err != nil {
		return RemoveFriendResult{}, err
	}

	var result RemoveFriendResult

	err = s.conn.TransactCtx(
		ctx,
		func(ctx context.Context, s sqlx.Session) error {
			var friendship struct {
				ID      int64  `db:"id"`
				Status  int    `db:"status"`
				Version uint64 `db:"version"`
			}

			err := s.QueryRowCtx(
				ctx,
				&friendship,
				`SELECT id, status, version
				 FROM friendships
				 WHERE user_a = ? AND user_b = ?
				 FOR UPDATE`,
				userA,
				userB,
			)
			if errors.Is(err, sqlx.ErrNotFound) {
				return ErrFriendshipNotFound
			}
			if err != nil {
				return fmt.Errorf("lock friendship: %w", err)
			}
			if friendship.Status != 1 {
				return ErrFriendshipNotFound
			}

			newVersion := friendship.Version + 1
			now := time.Now().UTC()

			_, err = s.ExecCtx(
				ctx,
				`UPDATE friendships
				 SET status = 2,
				 	version = ?,
				 	updated_at = ?
				 WHERE id = ?`,
				newVersion,
				now,
				friendship.ID,
			)
			if err != nil {
				return fmt.Errorf("remove friendship: %w", err)
			}

			result.FriendshipID = friendship.ID
			result.Version = newVersion
			return nil
		},
	)
	if err != nil {
		return RemoveFriendResult{}, err
	}

	return result, nil
}

type FriendRequestItem struct {
	ID          int64     `db:"id"`
	RequesterID string    `db:"requester_id"`
	RecipientID string    `db:"recipient_id"`
	Message     string    `db:"message"`
	Status      int       `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
}

func (s *MySQLStore) ListIncomingFriendRequests(
	ctx context.Context,
	userID string,
) ([]FriendRequestItem, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("user id is required")
	}

	var result []FriendRequestItem

	err := s.conn.QueryRowsCtx(
		ctx,
		&result,
		`SELECT
			id,
			requester_id,
			recipient_id,
			message,
			status,
			created_at
		 FROM friend_requests
		 WHERE recipient_id = ?
		   AND status = 1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list friend requests: %w", err)
	}

	return result, nil
}
