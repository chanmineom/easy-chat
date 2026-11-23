package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ConversationMessageContext struct {
	ChatType model.ChatType
	PeerID   string
}

func (s *MongoStore) GetConversationMessageContext(
	ctx context.Context,
	conversationID string,
	senderID string,
) (ConversationMessageContext, error) {
	var conversation model.Conversation
	err := s.convs.FindOne(
		ctx,
		bson.M{
			"_id": conversationID,
		},
	).Decode(&conversation)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ConversationMessageContext{}, ErrConversationNotFound
	}
	if err != nil {
		return ConversationMessageContext{}, err
	}
	if conversation.Dissolved {
		return ConversationMessageContext{}, ErrConversationDissolved
	}

	var sender model.ConversationMember
	err = s.convMember.FindOne(
		ctx,
		bson.M{
			"conversationId": conversationID,
			"userId":         senderID,
			"isVisible":      true,
		},
	).Decode(&sender)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ConversationMessageContext{}, ErrSenderNotMember
	}
	if err != nil {
		return ConversationMessageContext{}, err
	}

	result := ConversationMessageContext{
		ChatType: conversation.ChatType,
	}

	if conversation.ChatType != model.ChatTypeSingle {
		return result, nil
	}

	var peer model.ConversationMember
	err = s.convMember.FindOne(
		ctx,
		bson.M{
			"conversationId": conversationID,
			"userId": bson.M{
				"$ne": senderID,
			},
			"isVisible": true,
		},
	).Decode(&peer)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ConversationMessageContext{}, fmt.Errorf(
			"%w: direct conversation peer not found",
			ErrSenderNotMember,
		)
	}
	if err != nil {
		return ConversationMessageContext{}, err
	}

	result.PeerID = peer.UserID
	return result, nil
}

func (s *MongoStore) GetMessageBySenderAndClientID(ctx context.Context, senderID string, clientMessageID string) (model.Message, error) {
	return s.findMessageByClientID(
		ctx,
		senderID,
		clientMessageID,
	)
}
