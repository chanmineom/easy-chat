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

type ConversationSummary struct {
	ConversationID string
	ChatType       model.ChatType
	LastSeq        int64
	ReadSeq        int64
	DeliveredSeq   int64
	UnreadCount    int64
	LastActivityAt time.Time
	LastMessage    *model.Message
}

func (s *MongoStore) ListConversationSummaries(ctx context.Context, userID string, limit int, beforeActivityAt int64, beforeConversationID string) ([]ConversationSummary, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	filter := bson.M{
		"userId":    userID,
		"isVisible": true,
	}

	if beforeActivityAt > 0 && beforeConversationID != "" {
		cursorTime := time.UnixMilli(beforeActivityAt).UTC()
		filter["$or"] = bson.A{
			bson.M{
				"lastActivityAt": bson.M{"$lt": cursorTime},
			},
			bson.M{
				"lastActivityAt": cursorTime,
				"conversationId": bson.M{"$lt": beforeConversationID},
			},
		}
	}

	cursor, err := s.convMember.Find(
		ctx,
		filter,
		options.Find().SetSort(bson.D{
			{Key: "lastActivityAt", Value: -1},
			{Key: "conversationId", Value: -1},
		}).SetLimit(int64(limit+1)),
	)
	if err != nil {
		return nil, false, fmt.Errorf("find conversation members: %w", err)
	}
	defer cursor.Close(ctx)

	var members []model.ConversationMember
	if err := cursor.All(ctx, &members); err != nil {
		return nil, false, fmt.Errorf("decode conversation members: %w", err)
	}

	hasMore := len(members) > limit
	if hasMore {
		members = members[:limit]
	}
	if len(members) == 0 {
		return []ConversationSummary{}, false, nil
	}

	conversationIDs := make([]string, 0, len(members))
	for _, member := range members {
		conversationIDs = append(conversationIDs, member.ConversationID)
	}

	conversationMap, err := s.loadConversations(ctx, conversationIDs)
	if err != nil {
		return nil, false, err
	}

	summaries := make([]ConversationSummary, 0, len(members))
	for _, member := range members {
		conversation, ok := conversationMap[member.ConversationID]
		if !ok {
			continue
		}

		lastMessage, err := s.latestMessage(ctx, member.ConversationID)
		if err != nil {
			return nil, false, err
		}

		unreadCount, err := s.messages.CountDocuments(
			ctx,
			bson.M{
				"conversationId": member.ConversationID,
				"seq": bson.M{
					"$gt": member.ReadSeq,
				},
				"senderId": bson.M{
					"$ne": userID,
				},
			},
		)
		if err != nil {
			return nil, false, fmt.Errorf("count unread messages: %w", err)
		}

		summaries = append(summaries, ConversationSummary{
			ConversationID: member.ConversationID,
			ChatType:       conversation.ChatType,
			LastSeq:        conversation.LastSeq,
			ReadSeq:        member.ReadSeq,
			DeliveredSeq:   member.DeliveredSeq,
			UnreadCount:    unreadCount,
			LastActivityAt: member.LastActivityAt,
			LastMessage:    lastMessage,
		})
	}

	return summaries, hasMore, nil
}

func (s *MongoStore) loadConversations(ctx context.Context, conversationIDs []string) (map[string]model.Conversation, error) {
	cursor, err := s.convs.Find(ctx, bson.M{
		"_id": bson.M{"$in": conversationIDs},
	})
	if err != nil {
		return nil, fmt.Errorf("find conversations: %w", err)
	}
	defer cursor.Close(ctx)

	var conversations []model.Conversation
	if err := cursor.All(ctx, &conversations); err != nil {
		return nil, fmt.Errorf("decode conversations: %w", err)
	}

	result := make(map[string]model.Conversation, len(conversations))
	for _, conversation := range conversations {
		result[conversation.ID] = conversation
	}

	return result, nil
}

func (s *MongoStore) latestMessage(ctx context.Context, conversationID string) (*model.Message, error) {
	var message model.Message
	err := s.messages.FindOne(
		ctx,
		bson.M{
			"conversationId": conversationID,
		},
		options.FindOne().SetSort(bson.D{
			{Key: "seq", Value: -1},
		}),
	).Decode(&message)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find latest message: %w", err)
	}

	return &message, nil
}
