package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// MONGO_TEST_URL="$MONGO_URL" \
//   go test ./apps/im/repository \
//   -run TestAppendMessageIsOrderedAndIdempotent \
//   -v
func TestAppendMessageIsOrderedAndIdempotent(t *testing.T) {
	uri := os.Getenv("MONGO_TEST_URL")
	if uri == "" {
		t.Skip("MONGO_TEST_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	database := fmt.Sprintf("easy_chat_test_%d", time.Now().UnixNano())
	store, err := NewMongoStore(ctx, uri, database)
	if err != nil {
		t.Fatalf("NewMongoStore() error = %v", err)
	}
	defer func() {
		// _ = store.database.Drop(context.Background())
		_ = store.Close(context.Background())
	}()

	conversation, err := store.EnsureDirectConversation(ctx, "test-alice", "test-bob")
	if err != nil {
		t.Fatalf("EnsureDirectConversation() error = %v", err)
	}

	first, err := store.AppendMessage(ctx, AppendMessageInput{
		ConversationID:  conversation.ID,
		SenderID:        "test-alice",
		ClientMessageID: "client-1",
		ContentType:     "text",
		Content:         "hello",
	})
	if err != nil {
		t.Fatalf("first AppendMessage() error = %v", err)
	}

	second, err := store.AppendMessage(ctx, AppendMessageInput{
		ConversationID:  conversation.ID,
		SenderID:        "test-alice",
		ClientMessageID: "client-2",
		ContentType:     "text",
		Content:         "world",
	})
	if err != nil {
		t.Fatalf("second AppendMessage() error = %v", err)
	}

	retry, err := store.AppendMessage(ctx, AppendMessageInput{
		ConversationID:  conversation.ID,
		SenderID:        "test-alice",
		ClientMessageID: "client-1",
		ContentType:     "text",
		Content:         "hello",
	})
	if err != nil {
		t.Fatalf("retry AppendMessage() error = %v", err)
	}

	if first.Message.Sequence != 1 || second.Message.Sequence != 2 {
		t.Fatalf(
			"sequences = [%d, %d], want [1, 2]",
			first.Message.Sequence,
			second.Message.Sequence,
		)
	}
	if !retry.Duplicate {
		t.Fatal("retry was not reported as duplicate")
	}
	if retry.Message.ID != first.Message.ID {
		t.Fatalf(
			"retry message id = %s, want %s",
			retry.Message.ID.Hex(),
			first.Message.ID.Hex(),
		)
	}

	_, err = store.AppendMessage(ctx, AppendMessageInput{
		ConversationID:  conversation.ID,
		SenderID:        "test-alice",
		ClientMessageID: "client-1",
		ContentType:     "text",
		Content:         "changed",
	})
	if !errors.Is(err, ErrMessageIDConflict) {
		t.Fatalf("conflicting retry error = %v, want ErrMessageIDConflict", err)
	}

	messageCount, err := store.messages.CountDocuments(ctx, bson.M{
		"conversationId": conversation.ID,
	})
	if err != nil {
		t.Fatalf("count messages error = %v", err)
	}
	if messageCount != 2 {
		t.Fatalf("message count = %d, want 2", messageCount)
	}

	outboxCount, err := store.outbox.CountDocuments(ctx, bson.M{
		"partitionKey": conversation.ID,
	})
	if err != nil {
		t.Fatalf("count outbox error = %v", err)
	}
	if outboxCount != 2 {
		t.Fatalf("outbox count = %d, want 2", outboxCount)
	}
}
