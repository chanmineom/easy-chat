package outbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"github.com/chanmineom/easy-chat/apps/im/repository"
)

type fakePublisher struct {
	err    error
	events []model.OutboxEvent
}

func (p *fakePublisher) Publish(_ context.Context, event model.OutboxEvent) error {
	if p.err != nil {
		return p.err
	}
	p.events = append(p.events, event)
	return nil
}

func (p *fakePublisher) Close() error {
	return nil
}

func TestWorkerMarksPublishedEventSent(t *testing.T) {
	store, conversationID := prepareOutboxTest(t)
	defer dropOutboxTestStore(t, store)

	_, err := store.AppendMessage(context.Background(), repository.AppendMessageInput{
		ConversationID:  conversationID,
		SenderID:        "outbox-alice",
		ClientMessageID: "outbox-message-1",
		ContentType:     model.ContentTypeText,
		Content:         "hello outbox",
	})
	if err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}

	publisher := &fakePublisher{}
	worker := NewWorker(store, publisher)

	processed, err := worker.PublishNext(context.Background())
	if err != nil {
		t.Fatalf("PublishNext() error = %v", err)
	}
	if !processed {
		t.Fatal("worker did not claim an event")
	}
	if len(publisher.events) != 1 {
		t.Fatalf("published events = %d, want 1", len(publisher.events))
	}

	sent, err := store.GetOutboxEvent(context.Background(), publisher.events[0].EventID)
	if err != nil {
		t.Fatalf("GetOutboxEvent() error = %v", err)
	}
	if sent.Status != repository.OutboxSent {
		t.Fatalf("event status = %q, want %q", sent.Status, repository.OutboxSent)
	}
}

func TestWorkerReschedulesFailedEvent(t *testing.T) {
	store, conversationID := prepareOutboxTest(t)
	defer dropOutboxTestStore(t, store)

	_, err := store.AppendMessage(context.Background(), repository.AppendMessageInput{
		ConversationID:  conversationID,
		SenderID:        "outbox-alice",
		ClientMessageID: "outbox-message-2",
		ContentType:     model.ContentTypeText,
		Content:         "retry me",
	})
	if err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}

	worker := NewWorker(store, &fakePublisher{err: errors.New("kafka unavailable")})

	processed, err := worker.PublishNext(context.Background())
	if !processed {
		t.Fatal("worker did not claim an event")
	}
	if err == nil {
		t.Fatal("PublishNext() error = nil, want publish error")
	}

	event, found, err := store.ClaimNextOutboxEvent(
		context.Background(),
		time.Now().UTC().Add(2*time.Second),
		30*time.Second,
	)
	if err != nil {
		t.Fatalf("ClaimNextOutboxEvent() error = %v", err)
	}
	if !found {
		t.Fatal("failed event was not available for retry")
	}
	if event.Attempts != 2 {
		t.Fatalf("event attempts = %d, want 2", event.Attempts)
	}
}

func prepareOutboxTest(t *testing.T) (*repository.MongoStore, string) {
	t.Helper()

	uri := os.Getenv("MONGO_TEST_URL")
	if uri == "" {
		t.Skip("MONGO_TEST_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	database := fmt.Sprintf("easy_chat_outbox_test_%d", time.Now().UnixNano())
	store, err := repository.NewMongoStore(ctx, uri, database)
	if err != nil {
		t.Fatalf("NewMongoStore() error = %v", err)
	}

	conversation, err := store.EnsureDirectConversation(
		ctx,
		"outbox-alice",
		"outbox-bob",
	)
	if err != nil {
		_ = store.Close(context.Background())
		t.Fatalf("EnsureDirectConversation() error = %v", err)
	}

	return store, conversation.ID
}

func dropOutboxTestStore(t *testing.T, store *repository.MongoStore) {
	t.Helper()
	_ = store.Close(context.Background())
}
