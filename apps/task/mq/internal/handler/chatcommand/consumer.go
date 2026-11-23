package chatcommand

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/metrics"
	"github.com/chanmineom/easy-chat/apps/im/model"
	"github.com/chanmineom/easy-chat/apps/im/repository"
	gatewaypb "github.com/chanmineom/easy-chat/apps/im/ws/rpc/gateway"
	"github.com/chanmineom/easy-chat/apps/im/ws/rpc/gatewayclient"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"
	"github.com/chanmineom/easy-chat/apps/social/rpc/socialclient"
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/deadletter"
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/gatewayport"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var _ kq.ConsumeHandler = (*Consumer)(nil)

type Consumer struct {
	store       *repository.MongoStore
	gateway     gatewayport.Gateway
	Social      socialclient.Social
	deadLetter  deadletter.Publisher
	sourceTopic string
	sourceGroup string
}

func NewConsumer(store *repository.MongoStore, gateway gatewayport.Gateway, social socialclient.Social, deadLetter deadletter.Publisher, sourceTopic string, sourceGroup string) *Consumer {
	return &Consumer{
		store:       store,
		gateway:     gateway,
		Social:      social,
		deadLetter:  deadLetter,
		sourceTopic: sourceTopic,
		sourceGroup: sourceGroup,
	}
}

func (c *Consumer) Consume(ctx context.Context, key string, value string) error {
	logger := logx.WithContext(ctx)

	var command model.SendMessageCommand
	if err := json.Unmarshal([]byte(value), &command); err != nil {
		logger.Errorf("discard malformed chat command: %v", err)
		if dlqErr := c.publishDeadLetter(ctx, key, value, err); dlqErr != nil {
			return dlqErr
		}
		return nil
	}

	if command.CommandID == "" {
		logger.Error("discard chat command without command_id")
		cause := errors.New("chat command_id is required")
		if err := c.publishDeadLetter(
			ctx,
			key,
			value,
			cause,
		); err != nil {
			return err
		}

		return nil
	}

	if key != "" && key != command.ConversationID {
		logger.Errorf("discard chat command with mismatched kafka key: key=%s conversation=%s", key, command.ConversationID)
		cause := fmt.Errorf(
			"kafka key %q does not match conversation %q",
			key,
			command.ConversationID,
		)
		if err := c.publishDeadLetter(
			ctx,
			key,
			value,
			cause,
		); err != nil {
			return err
		}

		return nil
	}

	_, lookupErr := c.store.GetMessageBySenderAndClientID(
		ctx,
		command.SenderID,
		command.ClientMessageID,
	)

	needsFriendCheck := errors.Is(lookupErr, mongo.ErrNoDocuments)
	if lookupErr != nil && !needsFriendCheck {
		return lookupErr
	}

	if needsFriendCheck {
		conversationCtx, err := c.store.GetConversationMessageContext(
			ctx,
			command.ConversationID,
			command.SenderID,
		)
		if err != nil {
			if errors.Is(err, repository.ErrConversationNotFound) ||
				errors.Is(err, repository.ErrSenderNotMember) ||
				errors.Is(err, repository.ErrConversationDissolved) {
			} else {
				return err
			}
		}

		if err == nil && conversationCtx.ChatType == model.ChatTypeSingle {
			friendResp, err := c.Social.IsFriend(
				ctx,
				&social.IsFriendReq{
					UserA: command.SenderID,
					UserB: conversationCtx.PeerID,
				},
			)
			if err != nil {
				return err
			}

			if !friendResp.IsFriend {
				metrics.MessageRejectedTotal.Inc("not_friend")
				if err := c.pushRejection(
					ctx,
					command,
					"not_friend",
					"users are no longer friends",
				); err != nil {
					return err
				}
				return nil
			}
		}
	}

	result, err := c.store.AppendMessage(ctx, repository.AppendMessageInput{
		ConversationID:  command.ConversationID,
		SenderID:        command.SenderID,
		ClientMessageID: command.ClientMessageID,
		ContentType:     command.ContentType,
		Content:         command.Content,
	})
	if err != nil {
		// if permanentCommandError(err) {
		// 	logger.Errorf("discard rejected chat command command_id=%s: %v", command.CommandID, err)
		// 	return nil
		// }
		if code, message, rejected := commandRejection(err); rejected {
			if command.SenderID == "" {
				logger.Errorf(
					"rejected command has no sender command_id=%s: %v",
					command.CommandID,
					err,
				)

				cause := fmt.Errorf(
					"rejected command has no sender command_id=%s: %v",
					command.CommandID,
					err,
				)

				if dqlErr := c.publishDeadLetter(
					ctx,
					key,
					value,
					cause,
				); dqlErr != nil {
					return dqlErr
				}
				metrics.MessageRejectedTotal.Inc(code)
				return nil
			}

			if pushErr := c.pushRejection(ctx, command, code, message); pushErr != nil {
				return pushErr
			}

			logger.Infof(
				"chat command rejected command_id=%s sender=%s code=%s",
				command.CommandID,
				command.SenderID,
				code,
			)
			return nil
		}

		return err
	}

	if result.Duplicate {
		metrics.MessagePersistedTotal.Inc("duplicate")
		logger.Infof("chat command already applied command_id=%s message_id=%s seq=%d", command.CommandID, result.Message.ID.Hex(), result.Message.Sequence)
		return nil
	}

	metrics.MessagePersistedTotal.Inc("created")
	logger.Infof("message persisted command_id=%s message_id=%s conversation=%s seq=%d", command.CommandID, result.Message.ID.Hex(), result.Message.ConversationID, result.Message.Sequence)
	return nil
}

// func permanentCommandError(err error) bool {
// 	return errors.Is(err, repository.ErrConversationNotFound) ||
// 		errors.Is(err, repository.ErrSenderNotMember) ||
// 		errors.Is(err, repository.ErrMessageIDConflict) ||
// 		errors.Is(err, repository.ErrInvalidMessage)
// }

func (c *Consumer) pushRejection(ctx context.Context, command model.SendMessageCommand, code string, message string) error {
	_, err := c.gateway.PushCommandResult(ctx, &gatewaypb.PushCommandResultReq{
		UserId: command.SenderID,
		Result: &gatewayclient.CommandResult{
			CommandId:       command.CommandID,
			ConversationId:  command.ConversationID,
			ClientMessageId: command.ClientMessageID,
			Code:            code,
			Message:         message,
		},
	})
	if err != nil {
		return fmt.Errorf("push command rejection: %w", err)
	}

	return nil
}

func commandRejection(err error) (code string, message string, ok bool) {
	switch {
	case errors.Is(err, repository.ErrConversationNotFound):
		return "conversation_not_found",
			"conversation does not exist",
			true

	case errors.Is(err, repository.ErrSenderNotMember):
		return "not_conversation_member",
			"sender is not a conversation member",
			true

	case errors.Is(err, repository.ErrMessageIDConflict):
		return "client_message_id_conflict",
			"client_message_id was already used for another message",
			true

	case errors.Is(err, repository.ErrInvalidMessage):
		return "invalid_message",
			"message data is invalid",
			true

	case errors.Is(err, repository.ErrConversationDissolved):
		return "conversation_dissolved",
			"conversation has been dissolved",
			true

	default:
		return "", "", false
	}
}

func (c *Consumer) publishDeadLetter(
	ctx context.Context,
	key string,
	value string,
	cause error,
) error {
	message := deadletter.Message{
		SourceTopic: c.sourceTopic,
		ConsumerGroup: c.sourceGroup,
		Key:         key,
		Value:       value,
		Error:       cause.Error(),
		FailedAt:    time.Now().UTC(),
	}

	if err := c.deadLetter.Publish(ctx, message); err != nil {
		return fmt.Errorf("publish chat command dead letter: %w", err)
	}

	return nil
}
