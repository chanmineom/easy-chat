package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrConversationAccessDenied = errors.New(
	"user is not a conversation member",
)

func (s *MongoStore) ListMessagesAfter(ctx context.Context, userID string, conversationID string, afterSeq int64, limit int) ([]model.Message, bool, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	var member model.ConversationMember
	err := s.convMember.FindOne(
		ctx,
		bson.M{
			"conversationId": conversationID,
			"userId":         userID,
			"isVisible":      true,
		},
	).Decode(&member)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, false, ErrConversationAccessDenied
	}
	if err != nil {
		return nil, false, fmt.Errorf("find conversation member: %w", err)
	}

	cursor, err := s.messages.Find(
		ctx,
		bson.M{
			"conversationId": conversationID,
			"seq": bson.M{
				"$gt": afterSeq,
			},
		},
		options.Find().SetSort(bson.D{
			{Key: "seq", Value: 1},
		}).SetLimit(int64(limit+1)),
	)
	if err != nil {
		return nil, false, fmt.Errorf("find messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []model.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, false, fmt.Errorf("decode messages: %w", err)
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	return messages, hasMore, nil
}

func (s *MongoStore) ListMessagesBefore(ctx context.Context, userID string, conversationID string, beforeSeq int64, limit int) ([]model.Message, bool, error) {
	if beforeSeq <= 0 {
		return nil, false, fmt.Errorf(
			"before seq must be positive",
		)
	}

	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	var member model.ConversationMember
	err := s.convMember.FindOne(ctx, bson.M{
		"conversationId": conversationID,
		"userId":         userID,
		"isVisible":      true,
	}).Decode(&member)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, false, ErrConversationAccessDenied
	}
	if err != nil {
		return nil, false, fmt.Errorf(
			"find conversation member: %w",
			err,
		)
	}

	cursor, err := s.messages.Find(ctx, bson.M{
		"conversationId": conversationID,
		"seq": bson.M{
			"$lt": beforeSeq,
		},
	},
		options.Find().
			SetSort(bson.D{
				{Key: "seq", Value: -1},
			}).
			SetLimit(int64(limit+1)))
	if err != nil {
		return nil, false, fmt.Errorf(
			"find older messages: %w",
			err,
		)
	}
	defer cursor.Close(ctx)

	var messages []model.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, false, fmt.Errorf(
			"decode older messages: %w",
			err,
		)
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}

	return messages, hasMore, nil
}
