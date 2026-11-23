package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrMessageIDConflict     = errors.New("client message id was reused for a different message")
	ErrSenderNotMember       = errors.New("sender is not a conversation member")
	ErrConversationNotFound  = errors.New("conversation not found")
	ErrInvalidMessage        = errors.New("invalid message")
	ErrConversationDissolved = errors.New("conversation has been dissolved")
)

type AppendMessageInput struct {
	ConversationID  string
	SenderID        string
	ClientMessageID string
	ContentType     string
	Content         string
}

type AppendMessageResult struct {
	Message   model.Message
	Duplicate bool
}

func (s *MongoStore) AppendMessage(ctx context.Context, input AppendMessageInput) (AppendMessageResult, error) {
	if err := validateAppendMessage(input); err != nil {
		return AppendMessageResult{}, err
	}

	session, err := s.client.StartSession()
	if err != nil {
		return AppendMessageResult{}, fmt.Errorf("start mongo session: %w", err)
	}
	defer session.EndSession(ctx)

	var result AppendMessageResult

	_, err = session.WithTransaction(ctx, func(ctx context.Context) (any, error) {
		existing, err := s.findMessageByClientID(ctx, input.SenderID, input.ClientMessageID)
		if err == nil {
			if !sameMessageIntent(existing, input) {
				return nil, ErrMessageIDConflict
			}
			result = AppendMessageResult{
				Message:   existing,
				Duplicate: true,
			}
			return nil, nil
		}
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, err
		}

		var member model.ConversationMember
		err = s.convMember.FindOne(ctx, bson.M{
			"conversationId": input.ConversationID,
			"userId":         input.SenderID,
			"isVisible":      true,
		}).Decode(&member)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrSenderNotMember
		}
		if err != nil {
			return nil, fmt.Errorf("find sender membership: %w", err)
		}

		now := time.Now().UTC()

		var conversation model.Conversation
		err = s.convs.FindOneAndUpdate(
			ctx,
			bson.M{"_id": input.ConversationID},
			bson.M{
				"$inc": bson.M{"lastSeq": int64(1)},
				"$set": bson.M{"updatedAt": now},
			},
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		).Decode(&conversation)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrConversationNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("allocate conversation sequence: %w", err)
		}

		if conversation.Dissolved {
			return nil, ErrConversationDissolved
		}

		message := model.Message{
			ID:              bson.NewObjectID(),
			ConversationID:  input.ConversationID,
			Sequence:        conversation.LastSeq,
			SenderID:        input.SenderID,
			ClientMessageID: input.ClientMessageID,
			ContentType:     input.ContentType,
			Content:         input.Content,
			CreatedAt:       now,
		}

		if _, err := s.messages.InsertOne(ctx, message); err != nil {
			return nil, fmt.Errorf("insert message: %w", err)
		}

		_, err = s.convMember.UpdateMany(ctx, bson.M{
			"conversationId": message.ConversationID,
			"isVisible":      true,
		}, bson.M{
			"$max": bson.M{
				"lastActivityAt": now,
			},
			"$set": bson.M{
				"updatedAt": now,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("update conversation activity: %w", err)
		}

		updatedSender, err := s.convMember.UpdateOne(
			ctx,
			bson.M{
				"conversationId": message.ConversationID,
				"userId":         message.SenderID,
			},
			bson.M{
				"$set": bson.M{
					"readSeq":      message.Sequence,
					"deliveredSeq": message.Sequence,
					"updatedAt":    now,
				},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("update sender cursor: %w", err)
		}
		if updatedSender.MatchedCount == 0 {
			return nil, ErrSenderNotMember
		}

		event := model.MessageCreatedEvent{
			EventID:         "evt_" + message.ID.Hex(),
			MessageID:       message.ID.Hex(),
			ConversationID:  message.ConversationID,
			Sequence:        message.Sequence,
			SenderID:        message.SenderID,
			ClientMessageID: message.ClientMessageID,
			ContentType:     message.ContentType,
			Content:         message.Content,
			CreatedAt:       message.CreatedAt.UnixMilli(),
		}

		payload, err := json.Marshal(event)
		if err != nil {
			return nil, fmt.Errorf("marshal message event: %w", err)
		}

		outbox := model.OutboxEvent{
			ID:           bson.NewObjectID(),
			EventID:      event.EventID,
			Topic:        "im.message.created",
			PartitionKey: message.ConversationID,
			EventType:    "message.created",
			Payload:      payload,
			Status:       "pending",
			AvailableAt:  now,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if _, err := s.outbox.InsertOne(ctx, outbox); err != nil {
			return nil, fmt.Errorf("insert outbox event: %w", err)
		}

		result = AppendMessageResult{
			Message:   message,
			Duplicate: false,
		}

		return nil, nil
	})

	if err != nil {
		existing, lookupErr := s.findMessageByClientID(ctx, input.SenderID, input.ClientMessageID)
		if lookupErr == nil {
			if !sameMessageIntent(existing, input) {
				return AppendMessageResult{}, ErrMessageIDConflict
			}
			return AppendMessageResult{
				Message:   existing,
				Duplicate: true,
			}, nil
		}

		return AppendMessageResult{}, fmt.Errorf("append message transaction: %w", err)
	}

	return result, nil
}

func (s *MongoStore) findMessageByClientID(ctx context.Context, senderID string, clientMessageID string) (model.Message, error) {
	var message model.Message
	err := s.messages.FindOne(ctx, bson.M{
		"senderId":        senderID,
		"clientMessageId": clientMessageID,
	}).Decode(&message)

	return message, err
}

func sameMessageIntent(message model.Message, input AppendMessageInput) bool {
	return message.ConversationID == input.ConversationID && message.ContentType == input.ContentType && message.Content == input.Content
}

func validateAppendMessage(input AppendMessageInput) error {
	switch {
	case strings.TrimSpace(input.ConversationID) == "":
		return fmt.Errorf("%w: conversation id is required", ErrInvalidMessage)
	case strings.TrimSpace(input.SenderID) == "":
		return fmt.Errorf("%w: sender id is required", ErrInvalidMessage)
	case strings.TrimSpace(input.ClientMessageID) == "":
		return fmt.Errorf("%w: client message id is required", ErrInvalidMessage)
	case input.ContentType != model.ContentTypeText:
		return fmt.Errorf("%w: unsupported content type", ErrInvalidMessage)
	case strings.TrimSpace(input.Content) == "":
		return fmt.Errorf("%w: content is required", ErrInvalidMessage)
	case len(input.Content) > 32*1024:
		return fmt.Errorf("%w: content is too large", ErrInvalidMessage)
	default:
		return nil
	}
}
