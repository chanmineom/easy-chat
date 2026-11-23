package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-queue/kq"
)

type ChatPublisher interface {
	Publish(ctx context.Context, command model.SendMessageCommand) error
	Close() error
}

type KafkaChatPublisher struct {
	topic  string
	pusher *kq.Pusher
}

func NewKafkaChatPublisher(brokers []string, topic string) (*KafkaChatPublisher, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}
	if topic == "" {
		return nil, fmt.Errorf("chat command topic is required")
	}

	return &KafkaChatPublisher{
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

func (p *KafkaChatPublisher) Publish(ctx context.Context, command model.SendMessageCommand) error {
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

func (p *KafkaChatPublisher) Close() error {
	return p.pusher.Close()
}
