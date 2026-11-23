package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

const (
	SocialOutboxPending    = 1
	SocialOutboxProcessing = 2
	SocialOutboxSent       = 3
	SocialOutboxFailed     = 4
)

type SocialOutboxEvent struct {
	ID            uint64         `db:"id"`
	EventID       string         `db:"event_id"`
	AggregateType string         `db:"aggregate_type"`
	AggregateID   string         `db:"aggregate_id"`
	EventType     string         `db:"event_type"`
	EventVersion  uint64         `db:"event_version"`
	Payload       []byte         `db:"payload"`
	Status        int            `db:"status"`
	Attempts      int            `db:"attempts"`
	AvailableAt   time.Time      `db:"available_at"`
	LockedUntil   sql.NullString `db:"locked_until"`
	LastError     string         `db:"last_error"`
}

func (s *MySQLStore) ClaimNextOutboxEvent(ctx context.Context, now time.Time, lease time.Duration) (SocialOutboxEvent, bool, error) {
	var event SocialOutboxEvent
	var found bool

	err := s.conn.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		err := s.QueryRowCtx(
			ctx,
			&event,
			`SELECT
				id,
				event_id,
				aggregate_type,
				aggregate_id,
				event_type,
				event_version,
				payload,
				status,
				attempts,
				available_at,
				locked_until,
				last_error
			FROM social_outbox
			WHERE (
				status = ? AND available_at <= ?
			) OR (
			 	status = ? AND locked_until <= ?
			)
			ORDER BY id
			LIMIT 1
			FOR UPDATE SKIP LOCKED`,
			SocialOutboxPending,
			now,
			SocialOutboxProcessing,
			now,
		)
		if errors.Is(err, sqlx.ErrNotFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("select social outbox: %w", err)
		}

		_, err = s.ExecCtx(
			ctx,
			`UPDATE social_outbox
			SET status = ?,
				attempts = attempts + 1,
				locked_until = ?,
				updated_at = ?
			WHERE id = ?`,
			SocialOutboxProcessing,
			now.Add(lease),
			now,
			event.ID,
		)
		if err != nil {
			return fmt.Errorf("claim social outbox: %w", err)
		}

		event.Status = SocialOutboxProcessing
		event.Attempts++
		found = true

		return nil
	})
	if err != nil {
		return SocialOutboxEvent{}, false, err
	}

	return event, found, nil
}

func (s *MySQLStore) MarkOutboxSent(ctx context.Context, eventID string, now time.Time) error {
	result, err := s.conn.ExecCtx(
		ctx,
		`UPDATE social_outbox
		SET status = ?,
			sent_at = ?,
			locked_until = NULL,
			last_error = '',
			updated_at = ?
		WHERE event_id = ? AND status = ?`,
		SocialOutboxSent,
		now,
		now,
		eventID,
		SocialOutboxProcessing,
	)
	if err != nil {
		return fmt.Errorf("mark social outbox sent: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf(
			"mark social outbox sent: event %s is not leased",
			eventID,
		)
	}

	return nil
}

func (s *MySQLStore) MarkOutboxFailed(ctx context.Context, eventID string, retryAt time.Time, publishErr error, permanent bool) error {
	status := SocialOutboxPending
	if permanent {
		status = SocialOutboxFailed
	}

	message := "unknown sync error"
	if publishErr != nil {
		message = publishErr.Error()
	}
	if len(message) > 1000 {
		message = message[:1000]
	}

	result, err := s.conn.ExecCtx(
		ctx,
		`UPDATE social_outbox
		SET status = ?,
			available_at = ?,
			locked_until = NULL,
			last_error = ?,
			updated_at = ?
		WHERE event_id = ? AND status = ?`,
		status,
		retryAt,
		message,
		time.Now().UTC(),
		eventID,
		SocialOutboxProcessing,
	)
	if err != nil {
		return fmt.Errorf("mark social outbox failed: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf(
			"mark social outbox failed: event %s is not leased",
			eventID,
		)
	}

	return nil
}
