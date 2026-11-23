package repository

import (
	"context"
	"fmt"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *MongoStore) GetMessageByID(ctx context.Context, messageID string) (model.Message, error) {
	objectID, err := bson.ObjectIDFromHex(messageID)
	if err != nil {
		return model.Message{}, fmt.Errorf("invalid message id: %w", err)
	}

	var message model.Message
	if err := s.messages.FindOne(ctx, bson.M{
		"_id": objectID,
	}).Decode(&message); err != nil {
		return model.Message{}, err
	}

	return message, nil
}

func (s *MongoStore) ListConversationMemberIDs(ctx context.Context, conversationID string) ([]string, error) {
	cursor, err := s.convMember.Find(ctx, bson.M{
		"conversationId": conversationID,
		"isVisible":      true,
	}, options.Find().SetProjection(bson.M{
		"_id":    0,
		"userId": 1,
	}))
	if err != nil {
		return nil, fmt.Errorf("find conversation members: %w", err)
	}
	defer cursor.Close(ctx)

	var rows []struct {
		UserID string `bson:"userId"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("decode conversation members: %w", err)
	}

	userIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.UserID != "" {
			userIDs = append(userIDs, row.UserID)
		}
	}
	return userIDs, nil
}
