package deadletter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/metrics"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-queue/kq"
)

type Message struct {
	SourceTopic   string    `json:"source_topic"`
	ConsumerGroup string    `json:"consumer_group"`
	Key           string    `json:"key"`
	Value         string    `json:"value"`
	Error         string    `json:"error"`
	FailedAt      time.Time `json:"failed_at"`
}

type Publisher interface {
	Publish(ctx context.Context, message Message) error
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
		return nil, fmt.Errorf("dead letter topic is required")
	}

	return &KafkaPublisher{
		topic:  topic,
		pusher: kq.NewPusher(brokers, topic, kq.WithSyncPush(), kq.WithBalancer(&kafka.Hash{}), kq.WithAllowAutoTopicCreation()),
	}, nil
}

func (p *KafkaPublisher) Publish(ctx context.Context, message Message) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode dead letter: %w", err)
	}

	key := message.SourceTopic
	if message.Key != "" {
		key += ":" + message.Key
	}

	if err := p.pusher.PushWithKey(
		ctx,
		key,
		string(payload),
	); err != nil {
		return err
	}

	metrics.DeadLettersTotal.Inc(
		message.SourceTopic,
		message.ConsumerGroup,
	)
	return nil
}

func (p *KafkaPublisher) Close() error {
	return p.pusher.Close()
}
