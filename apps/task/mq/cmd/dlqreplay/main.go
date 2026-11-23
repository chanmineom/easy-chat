package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chanmineom/easy-chat/apps/task/mq/internal/deadletter"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-queue/kq"
)

var allowedSource = map[string]struct{}{
	"im.chat.commands":    {},
	"im.message.created":  {},
	"im.receipt.commands": {},
}

func main() {
	brokerDefault := os.Getenv("KAFKA_BROKER")
	if brokerDefault == "" {
		brokerDefault = "127.0.0.1:9092"
	}

	brokerList := flag.String(
		"brokers",
		brokerDefault,
		"comma-separated kafka brokers",
	)
	dlqTopic := flag.String(
		"topic",
		"im.dead.letter",
		"dead letter topic",
	)
	groupID := flag.String(
		"group",
		"easy-chat-dlq-replay",
		"replay consumer group",
	)
	limit := flag.Int(
		"limit",
		1,
		"maximum number of dead letters to inspect/replay",
	)
	execute := flag.Bool(
		"execute",
		false,
		"actually replay messages; default is dry-run",
	)
	timeout := flag.Duration(
		"timeout",
		30*time.Second,
		"maximum time to wait for messages",
	)
	flag.Parse()

	if *limit <= 0 {
		log.Fatal("-limit must be greater than zero")
	}

	brokers := splitBrokers(*brokerList)
	if len(brokers) == 0 {
		log.Fatal("at least one kafka broker is required")
	}
	if *dlqTopic == "" {
		log.Fatal("dead letter topic is required")
	}
	if *groupID == "" {
		log.Fatal("consumer group is required")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		*timeout,
	)
	defer cancel()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     *groupID,
		Topic:       *dlqTopic,
		StartOffset: kafka.FirstOffset,
		MinBytes:    1,
		MaxBytes:    10 * 1024 * 1024,
	})
	defer reader.Close()

	pushers := make(map[string]*kq.Pusher)
	defer func() {
		for _, pusher := range pushers {
			_ = pusher.Close()
		}
	}()

	processed := 0

	for processed < *limit {
		record, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				break
			}
			log.Fatalf("fetch dead letter: %v", err)
		}

		var message deadletter.Message
		if err := json.Unmarshal(record.Value, &message); err != nil {
			log.Fatalf(
				"dead letter value is invalid JSON offset=%d: %v",
				record.Offset,
				err,
			)
		}

		if _, ok := allowedSource[message.SourceTopic]; !ok {
			log.Fatalf(
				"source topic %q is not allowed offset=%d",
				message.SourceTopic,
				record.Offset,
			)
		}
		if message.Value == "" {
			log.Fatalf(
				"dead letter value is empty offset=%d",
				record.Offset,
			)
		}

		processed++

		fmt.Printf(
			"dead_letter offset=%d source=%s group=%s key=%s\n",
			record.Offset,
			message.SourceTopic,
			message.ConsumerGroup,
			message.Key,
		)
		fmt.Printf("original_error=%s\n", message.Error)
		fmt.Printf("value=%s\n", message.Value)

		if !*execute {
			fmt.Println("dry-run: message was not replayed")
			fmt.Println(strings.Repeat("-", 72))
			continue
		}

		pusher := pushers[message.SourceTopic]
		if pusher == nil {
			pusher = kq.NewPusher(
				brokers,
				message.SourceTopic,
				kq.WithSyncPush(),
				kq.WithBalancer(&kafka.Hash{}),
			)
			pushers[message.SourceTopic] = pusher
		}

		if err := pusher.PushWithKey(
			ctx,
			message.Key,
			message.Value,
		); err != nil {
			log.Fatalf(
				"replay message offset=%d: %v",
				record.Offset,
				err,
			)
		}

		if err := reader.CommitMessages(ctx, record); err != nil {
			log.Fatalf(
				"commit replayed dead letter offset=%d: %v",
				record.Offset,
				err,
			)
		}

		fmt.Println("replayed and committed")
		fmt.Println(strings.Repeat("-", 72))
	}

	if processed == 0 {
		fmt.Println("no dead letters found")
		return
	}

	fmt.Printf(
		"processed=%d execute=%t\n",
		processed,
		*execute,
	)
}

func splitBrokers(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}
