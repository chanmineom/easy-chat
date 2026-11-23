package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	ConversationCollection       = "conversations"
	ConversationMemberCollection = "conversation_members"
	MessageCollection            = "messages"
	OutboxCollection             = "outbox_events"
)

type MongoStore struct {
	client     *mongo.Client
	database   *mongo.Database
	convs      *mongo.Collection
	convMember *mongo.Collection
	messages   *mongo.Collection
	outbox     *mongo.Collection
}

func NewMongoStore(ctx context.Context, uri, database string) (*MongoStore, error) {
	if uri == "" {
		return nil, fmt.Errorf("mongo uri is required")
	}
	if database == "" {
		return nil, fmt.Errorf("mongo database is required")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	store := &MongoStore{
		client:     client,
		database:   client.Database(database),
		convs:      client.Database(database).Collection(ConversationCollection),
		convMember: client.Database(database).Collection(ConversationMemberCollection),
		messages:   client.Database(database).Collection(MessageCollection),
		outbox:     client.Database(database).Collection(OutboxCollection),
	}

	if err := store.ensureIndexes(ctx); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}

	return store, nil
}

func (s *MongoStore) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

func (s *MongoStore) ensureIndexes(ctx context.Context) error {
	_, err := s.convs.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "directKey", Value: 1},
			},
			Options: options.Index().SetName("uk_conversation_direct_key").SetUnique(true).SetPartialFilterExpression(bson.M{
				"directKey": bson.M{
					"$type": "string",
				},
			}),
		},
		{
			Keys: bson.D{
				{
					Key: "updatedAt", Value: -1,
				},
			},
			Options: options.Index().SetName("idx_conversation_updated_at"),
		},
	})
	if err != nil {
		return fmt.Errorf("create conversation indexes: %w", err)
	}

	_, err = s.convMember.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "conversationId", Value: 1},
				{Key: "userId", Value: 1},
			},
			Options: options.Index().SetName("uk_conversation_member").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "isVisible", Value: 1},
				{Key: "updatedAt", Value: -1},
			},
			Options: options.Index().SetName("idx_member_user_visible_updated"),
		},
	})
	if err != nil {
		return fmt.Errorf("create conversation member indexes: %w", err)
	}

	_, err = s.messages.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "senderId", Value: 1},
				{Key: "clientMessageId", Value: 1},
			},
			Options: options.Index().SetName("uk_message_client_id").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "conversationId", Value: 1},
				{Key: "seq", Value: 1},
			},
			Options: options.Index().SetName("uk_message_conversation_seq").SetUnique(true),
		},
	})
	if err != nil {
		return fmt.Errorf("create message indexes: %w", err)
	}

	_, err = s.outbox.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "eventId", Value: 1},
			},
			Options: options.Index().SetName("uk_outbox_event_id").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "availableAt", Value: 1},
			},
			Options: options.Index().SetName("idx_outbox_pending"),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "lockedUntil", Value: 1},
			},
			Options: options.Index().SetName("idx_outbox_locked"),
		},
		{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "isVisible", Value: 1},
				{Key: "lastActivityAt", Value: -1},
				{Key: "conversationId", Value: -1},
			},
			Options: options.Index().SetName("idx_member_user_activity"),
		},
	})
	if err != nil {
		return fmt.Errorf("create outbox indexes: %w", err)
	}

	return nil
}
