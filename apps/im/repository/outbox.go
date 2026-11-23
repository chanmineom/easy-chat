package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	OutboxPending    = "pending"
	OutboxPublishing = "publishing"
	OutboxSent       = "sent"
)

// ClaimNextOutboxEvent 以原子方式将一个事件租约授予当前 worker。处于
// pending 状态的事件可立即被领取；worker 崩溃后，已过期的发布租约可以被
// 重新收回（恢复）。
func (s *MongoStore) ClaimNextOutboxEvent(ctx context.Context, now time.Time, lease time.Duration) (model.OutboxEvent, bool, error) {
	filter := bson.M{
		"$or": bson.A{
			bson.M{
				"status": OutboxPending,
				"availableAt": bson.M{
					"$lte": now,
				},
			},
			bson.M{
				"status": OutboxPublishing,
				"lockedUntil": bson.M{
					"$lte": now,
				},
			},
		},
	}

	update := bson.M{
		"$set": bson.M{
			"status":      OutboxPublishing,
			"lockedUntil": now.Add(lease),
			"updatedAt":   now,
		},
		"$inc": bson.M{
			"attempts": 1,
		},
	}

	var event model.OutboxEvent

	err := s.outbox.FindOneAndUpdate(
		ctx,
		filter,
		update,
		options.FindOneAndUpdate().SetSort(bson.D{
			{Key: "availableAt", Value: 1},
		}).SetReturnDocument(options.After),
	).Decode(&event)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return model.OutboxEvent{}, false, nil
	}
	if err != nil {
		return model.OutboxEvent{}, false, fmt.Errorf("claim outbox event: %w", err)
	}

	return event, true, nil
}

func (s *MongoStore) MarkOutboxSent(ctx context.Context, eventID string, now time.Time) error {
	result, err := s.outbox.UpdateOne(
		ctx,
		bson.M{
			"eventId": eventID,
			"status":  OutboxPublishing,
		},
		bson.M{
			"$set": bson.M{
				"status":    OutboxSent,
				"sentAt":    now,
				"updatedAt": now,
			},
			"$unset": bson.M{
				"lockedUntil": "",
				"lastError":   "",
			},
		},
	)
	if err != nil {
		return fmt.Errorf("mark outbox sent: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("mark outbox sent: event %s is not leased", eventID)
	}

	return nil
}

func (s *MongoStore) MarkOutboxFailed(ctx context.Context, eventID string, retryAt time.Time, publishErr error) error {
	message := "unknown publish error"
	if publishErr != nil {
		message = publishErr.Error()
	}

	result, err := s.outbox.UpdateOne(ctx, bson.M{
		"eventId": eventID,
		"status":  OutboxPublishing,
	},
		bson.M{
			"$set": bson.M{
				"status":      OutboxPending,
				"availableAt": retryAt,
				"lastError":   message,
				"updatedAt":   time.Now().UTC(),
			},
			"$unset": bson.M{
				"lockedUntil": "",
			},
		})
	if err != nil {
		return fmt.Errorf("mark outbox failed: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("mark outbox failed: event %s is not leased", eventID)
	}

	return nil
}

func (s *MongoStore) GetOutboxEvent(ctx context.Context, eventID string) (model.OutboxEvent, error) {
	var event model.OutboxEvent
	err := s.outbox.FindOne(ctx, bson.M{
		"eventId": eventID,
	}).Decode(&event)
	if err != nil {
		return model.OutboxEvent{}, err
	}

	return event, nil
}

func (s *MongoStore) CountPendingOutbox(ctx context.Context) (int64, error) {
	return s.outbox.CountDocuments(
		ctx,
		bson.M{
			"status": bson.M{
				"$in": bson.A{
					OutboxPending,
					OutboxPublishing,
				},
			},
		},
	)
}
