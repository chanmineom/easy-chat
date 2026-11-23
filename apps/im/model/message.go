package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const ContentTypeText = "text"

type Message struct {
	ID              bson.ObjectID `bson:"_id"`
	ConversationID  string        `bson:"conversationId"`
	Sequence        int64         `bson:"seq"`
	SenderID        string        `bson:"senderId"`
	ClientMessageID string        `bson:"clientMessageId"`
	ContentType     string        `bson:"contentType"`
	Content         string        `bson:"content"`
	CreatedAt       time.Time     `bson:"createdAt"`
}

// MessageCreatedEvent 之后会被序列化到 Kafka 中。它会先存储到 MongoDB，这样即使 broker 发生故障，也不会丢失一条已经持久化的消息。
type MessageCreatedEvent struct {
	EventID         string `json:"event_id"`
	MessageID       string `json:"message_id"`
	ConversationID  string `json:"conversation_id"`
	Sequence        int64  `json:"sequence"`
	SenderID        string `json:"sender_id"`
	ClientMessageID string `json:"client_message_id"`
	ContentType     string `json:"content_type"`
	Content         string `json:"content"`
	CreatedAt       int64  `json:"created_at"`
}

type OutboxEvent struct {
	ID           bson.ObjectID `bson:"_id"`
	EventID      string        `bson:"eventId"`
	Topic        string        `bson:"topic"`
	PartitionKey string        `bson:"partitionKey"`
	EventType    string        `bson:"eventType"`
	Payload      []byte        `bson:"payload"`

	Status      string    `bson:"status"`
	Attempts    int       `bson:"attempts"`
	AvailableAt time.Time `bson:"availableAt"`
	// lockedUntil 用于让多个发布器实例安全地争抢事件，防止同一条 Outbox 被多个 worker 同时发送。
	LockedUntil time.Time `bson:"lockedUntil,omitempty"`
	SentAt      time.Time `bson:"sentAt,omitempty"`
	LastError   string    `bson:"lastError,omitempty"`

	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}
