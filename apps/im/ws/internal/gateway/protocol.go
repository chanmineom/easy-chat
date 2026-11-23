package gateway

import "encoding/json"

const (
	IdentityClaim = "easy-chat-uid"

	TypePing             = "ping"
	TypePong             = "pong"
	TypeError            = "error"
	TypeMessageCreated   = "message.created"
	TypeMessageDelivered = "message.delivered"
	TypeMessageRead      = "message.read"
	TypeMessageSend      = "message.send"
	TypeMessageAccepted  = "message.accepted"
	TypeMessageRejected  = "message.rejected"
	TypeReceiptAccepted  = "receipt.accepted"
	TypeConnectionReady  = "connection.ready"
)

type Envelope struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Error     string          `json:"error,omitempty"`
}

func NewDataEnvelope(kind string, data any) (Envelope, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		Type: kind,
		Data: payload,
	}, nil
}
