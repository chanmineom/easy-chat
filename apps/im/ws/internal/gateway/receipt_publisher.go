package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-queue/kq"
)

type ReceiptPublisher interface {
	Publish(ctx context.Context, command model.ReceiptCommand) error
	Close() error
}

type KafkaReceiptPublisher struct {
	topic  string
	pusher *kq.Pusher
}

func NewKafkaReceiptPusher(brokers []string, topic string) (*KafkaReceiptPublisher, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}
	if topic == "" {
		return nil, fmt.Errorf("receipt command topic is required")
	}

	return &KafkaReceiptPublisher{
		topic: topic,
		pusher: kq.NewPusher(
			brokers,
			topic,
			kq.WithSyncPush(),
			kq.WithBalancer(&kafka.Hash{}),
			kq.WithAllowAutoTopicCreation(),
		),
	}, nil
}

func (p *KafkaReceiptPublisher) Publish(ctx context.Context, command model.ReceiptCommand) error {
	payload, err := json.Marshal(command)
	if err != nil {
		return err
	}

	return p.pusher.PushWithKey(
		ctx,
		command.ConversationID,
		string(payload),
	)
}

func (p *KafkaReceiptPublisher) Close() error {
	return p.pusher.Close()
}
