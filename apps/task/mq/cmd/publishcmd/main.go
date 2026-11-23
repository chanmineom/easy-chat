package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-queue/kq"
)

func main() {
	brokerDefault := os.Getenv("KAFKA_BROKER")
	if brokerDefault == "" {
		brokerDefault = "127.0.0.1:9092"
	}

	broker := flag.String("broker", brokerDefault, "kafka broker")
	topic := flag.String("topic", "im.chat.commands", "kafka topic")
	conversationID := flag.String("conversation", "", "conversation id")
	senderID := flag.String("sender", "", "sender id")
	clientMessageID := flag.String("client-message", "", "client message id")
	content := flag.String("content", "", "message content")
	flag.Parse()

	command := model.SendMessageCommand{
		CommandID:       "cmd-" + time.Now().Format("20060102150405.000000000"),
		ConversationID:  *conversationID,
		SenderID:        *senderID,
		ClientMessageID: *clientMessageID,
		ContentType:     model.ContentTypeText,
		Content:         *content,
		RequestedAt:     time.Now().UnixMilli(),
	}

	payload, err := json.Marshal(command)
	if err != nil {
		log.Fatal(err)
	}

	pusher := kq.NewPusher(
		[]string{*broker},
		*topic,
		kq.WithSyncPush(),
		kq.WithBalancer(&kafka.Hash{}),
		kq.WithAllowAutoTopicCreation(),
	)
	defer pusher.Close()

	if err := pusher.PushWithKey(
		context.Background(),
		command.ConversationID,
		string(payload),
	); err != nil {
		log.Fatal(err)
	}

	log.Printf("published command: %s", command.CommandID)
}
