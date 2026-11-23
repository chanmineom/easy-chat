package receiptcommand

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/metrics"
	"github.com/chanmineom/easy-chat/apps/im/model"
	"github.com/chanmineom/easy-chat/apps/im/repository"
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/deadletter"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
)

var _ kq.ConsumeHandler = (*Consumer)(nil)

type Consumer struct {
	store       *repository.MongoStore
	deadLetter  deadletter.Publisher
	sourceTopic string
	sourceGroup string
}

func NewConsumer(
	store *repository.MongoStore,
	deadLetter deadletter.Publisher,
	sourceTopic string,
	sourceGroup string,
) *Consumer {
	return &Consumer{
		store:       store,
		deadLetter:  deadLetter,
		sourceTopic: sourceTopic,
		sourceGroup: sourceGroup,
	}
}

func (c *Consumer) Consume(ctx context.Context, key string, value string) error {
	logger := logx.WithContext(ctx)

	var command model.ReceiptCommand
	if err := json.Unmarshal([]byte(value), &command); err != nil {
		logger.Errorf("discard malformed receipt command: %v", err)

		cause := fmt.Errorf("decode receipt command: %w", err)

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

	if command.Type != model.ReceiptDelivered && command.Type != model.ReceiptRead {
		logger.Errorf(
			"discard receipt with unsupported type type=%s",
			command.Type,
		)

		cause := fmt.Errorf(
			"unsupported receipt type %q",
			command.Type,
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

	if command.UserID == "" || command.ConversationID == "" || command.Sequence <= 0 {
		cause := errors.New(
			"user_id, conversation_id and positive sequence are required",
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

	if key != "" && key != command.ConversationID {
		logger.Errorf(
			"discard receipt with mismatched kafka key key=%s conversation=%s",
			key,
			command.ConversationID,
		)
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

	err := c.store.AdvanceReceiptCursor(ctx, command.UserID, command.ConversationID, command.Sequence, command.Type)
	if err != nil {
		if permanentReceiptError(err) {
			metrics.ReceiptProcessedTotal.Inc(
				string(command.Type),
				"rejected",
			)
			logger.Errorf(
				"discard rejected receipt command_id=%s type=%s user=%s conversation=%s seq=%d: %v",
				command.CommandID,
				command.Type,
				command.UserID,
				command.ConversationID,
				command.Sequence,
				err,
			)
			return nil
		}
		
		metrics.ReceiptProcessedTotal.Inc(
			string(command.Type),
			"retryable_error",
		)
		return err
	}

	metrics.ReceiptProcessedTotal.Inc(
		string(command.Type),
		"accepted",
	)
	logger.Infof(
		"receipt accepted command_id=%s type=%s user=%s conversation=%s seq=%d",
		command.CommandID,
		command.Type,
		command.UserID,
		command.ConversationID,
		command.Sequence,
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
			"publish receipt dead letter: %w",
			err,
		)
	}

	return nil
}

func permanentReceiptError(err error) bool {
	return errors.Is(err, repository.ErrConversationNotFound) ||
		errors.Is(err, repository.ErrReceiptAhead) ||
		errors.Is(err, repository.ErrReceiptMemberNotFound)
}
