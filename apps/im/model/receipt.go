package model

type ReceiptType string

const (
	ReceiptDelivered ReceiptType = "message.delivered"
	ReceiptRead      ReceiptType = "message.read"
)

type ReceiptCommand struct {
	CommandID      string      `json:"command_id"`
	Type           ReceiptType `json:"type"`
	UserID         string      `json:"user_id"`
	ConversationID string      `json:"conversation_id"`
	Sequence       int64       `json:"sequence"`
	ObservedAt     int64       `json:"observed_at"`
}
