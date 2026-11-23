package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	ErrReceiptAhead          = errors.New("receipt sequence is ahead of conversation")
	ErrReceiptMemberNotFound = errors.New("receipt user is not a conversation member")
)

func (s *MongoStore) AdvanceReceiptCursor(ctx context.Context, userID string, conversationID string, sequence int64, receiptType model.ReceiptType) error {
	if sequence <= 0 {
		return fmt.Errorf("sequence must be positive")
	}

	var conversation model.Conversation
	err := s.convs.FindOne(ctx, bson.M{"_id": conversationID}).Decode(&conversation)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrConversationNotFound
	}
	if err != nil {
		return fmt.Errorf("find conversation: %w", err)
	}
	if sequence > conversation.LastSeq {
		return ErrReceiptAhead
	}

	now := time.Now().UTC()
	maxCursor := bson.M{
		"deliveredSeq": sequence,
	}
	if receiptType == model.ReceiptRead {
		maxCursor["readSeq"] = sequence
	}

	result, err := s.convMember.UpdateOne(ctx, bson.M{
		"conversationId": conversationID,
		"userId":         userID,
		"isVisible":      true,
	}, bson.M{
		"$max": maxCursor,
		"$set": bson.M{
			"updatedAt": now,
		},
	})
	if err != nil {
		return fmt.Errorf("update receipt cursor: %w", err)
	}
	if result.MatchedCount == 0 {
		return ErrReceiptMemberNotFound
	}

	return nil
}
