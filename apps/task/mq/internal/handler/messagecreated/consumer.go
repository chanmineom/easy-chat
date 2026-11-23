package messagecreated

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
	deadLetter  deadletter.Publisher
	sourceTopic string
	sourceGroup string
}

func NewConsumer(store *repository.MongoStore, gateway gatewayport.Gateway, deadLetter deadletter.Publisher, sourceTopic string, sourceGroup string) *Consumer {
	return &Consumer{
		store:       store,
		gateway:     gateway,
		deadLetter:  deadLetter,
		sourceTopic: sourceTopic,
		sourceGroup: sourceGroup,
	}
}

func (c *Consumer) Consume(ctx context.Context, key string, value string) error {
	logger := logx.WithContext(ctx)

	var event model.MessageCreatedEvent
	if err := json.Unmarshal([]byte(value), &event); err != nil {
		logger.Errorf("discard malformed message event: %v", err)
		cause := fmt.Errorf("decode message event: %w", err)

		if dlqErr := c.publishDeadLetter(
			ctx,
			key,
			value,
			cause,
		); dlqErr != nil {
			return dlqErr
		}
		return nil
	}

	if event.EventID == "" || event.MessageID == "" {
		logger.Error("discard message event without ids")

		cause := errors.New("event_id and message_id are required")

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

	if key != "" && key != event.ConversationID {
		logger.Errorf(
			"discard message event with mismatched kafka key: key=%s conversation=%s",
			key,
			event.ConversationID,
		)

		cause := fmt.Errorf(
			"kafka key %q does not match conversation %q",
			key,
			event.ConversationID,
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

	message, err := c.store.GetMessageByID(ctx, event.MessageID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.Errorf(
				"discard message event for missing message message_id=%s",
				event.MessageID,
			)

			cause := fmt.Errorf(
				"persisted message does not exist message_id=%s",
				event.MessageID,
			)

			if dlqErr := c.publishDeadLetter(
				ctx,
				key,
				value,
				cause,
			); dlqErr != nil {
				return dlqErr
			}

			return nil
		}
		return err
	}

	if !eventMatchesMessage(event, message) {
		logger.Errorf(
			"discard inconsistent message event event_id=%s message_id=%s",
			event.EventID,
			event.MessageID,
		)
		cause := fmt.Errorf(
			"event fields do not match persisted message message_id=%s",
			event.MessageID,
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

	userIDs, err := c.store.ListConversationMemberIDs(ctx, message.ConversationID)
	if err != nil {
		return err
	}
	if len(userIDs) == 0 {
		metrics.MessagePushTotal.Inc("no_visible_recipients")
		logger.Errorf(
			"message has no visible recipients message_id=%s conversation=%s",
			message.ID.Hex(),
			message.ConversationID,
		)
		return nil
	}

	response, err := c.gateway.PushToUsers(ctx, &gatewaypb.PushToUsersReq{
		UserIds: userIDs,
		Message: &gatewaypb.MessageData{
			MessageId:       message.ID.Hex(),
			ConversationId:  message.ConversationID,
			Sequence:        message.Sequence,
			SenderId:        message.SenderID,
			ClientMessageId: message.ClientMessageID,
			ContentType:     message.ContentType,
			Content:         message.Content,
			CreatedAt:       message.CreatedAt.UnixMilli(),
		},
	})
	if err != nil {
		metrics.MessagePushTotal.Inc("error")
		return err
	}

	// 推送成功
	metrics.MessagePushTotal.Inc("delivered")

	// 区分在线和离线
	// if response.DeliveredConnections == 0 {
	// 	metrics.MessagePushTotal.Inc("offset")
	// } else {
	// 	metrics.MessagePushTotal.Inc("delivered")
	// }

	logger.Info(
		"message pushed message_id=%s conversation=%s seq=%d online_users=%d delivered_connections=%d",
		message.ID.Hex(),
		message.ConversationID,
		message.Sequence,
		response.OnlineUsers,
		response.DeliveredConnections,
	)

	return nil
}

func (c *Consumer) publishDeadLetter(
	ctx context.Context,
	key string,
	value string,
	cause error,
) error {
	err := c.deadLetter.Publish(
		ctx,
		deadletter.Message{
			SourceTopic:   c.sourceTopic,
			ConsumerGroup: c.sourceGroup,
			Key:           key,
			Value:         value,
			Error:         cause.Error(),
			FailedAt:      time.Now().UTC(),
		},
	)
	if err != nil {
		return fmt.Errorf(
			"publish message event dead letter: %w",
			err,
		)
	}

	return nil
}

func eventMatchesMessage(event model.MessageCreatedEvent, message model.Message) bool {
	return event.MessageID == message.ID.Hex() &&
		event.ConversationID == message.ConversationID &&
		event.Sequence == message.Sequence &&
		event.SenderID == message.SenderID &&
		event.ClientMessageID == message.ClientMessageID &&
		event.ContentType == message.ContentType &&
		event.Content == message.Content
}
