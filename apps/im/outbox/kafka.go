package outbox

import (
	"context"
	"fmt"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-queue/kq"
)

type EventPubliser interface {
	Publish(ctx context.Context, event model.OutboxEvent) error
	Close() error
}

type KafkaPublisher struct {
	topic  string
	pusher *kq.Pusher
}

func NewKafkaPublisher(brokers []string, topic string) (*KafkaPublisher, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}
	if topic == "" {
		return nil, fmt.Errorf("kafka topic is required")
	}

	return &KafkaPublisher{
		topic:  topic,
		pusher: kq.NewPusher(brokers, topic, kq.WithSyncPush(), kq.WithBalancer(&kafka.Hash{}), kq.WithAllowAutoTopicCreation()),
	}, nil
}

func (p *KafkaPublisher) Publish(ctx context.Context, event model.OutboxEvent) error {
	if event.Topic != p.topic {
		return fmt.Errorf("event topic %q does not match publisher topic %q", event.Topic, p.topic)
	}

	return p.pusher.PushWithKey(ctx, event.PartitionKey, string(event.Payload))
}

func (p *KafkaPublisher) Close() error {
	return p.pusher.Close()
}
