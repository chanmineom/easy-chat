package gateway

import (
	"context"
	"encoding/json"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/metrics"
	"github.com/chanmineom/easy-chat/apps/im/model"
)

type ReceiptData struct {
	ConversationID string `json:"conversation_id"`
	Sequence       int64  `json:"sequence"`
}

func (h *ConnectHandler) handleReceipt(client *Client, message Envelope, receiptType model.ReceiptType) {
	var data ReceiptData
	if err := json.Unmarshal(message.Data, &data); err != nil {
		client.sendError(message.RequestID, "invalid receipt data")
		return
	}
	if data.ConversationID == "" || data.Sequence <= 0 {
		client.sendError(message.RequestID, "conversation_id and positive sequence are required")
		return
	}

	command := model.ReceiptCommand{
		CommandID:      newSessionID(),
		Type:           receiptType,
		UserID:         client.UserID,
		ConversationID: data.ConversationID,
		Sequence:       data.Sequence,
		ObservedAt:     time.Now().UnixMilli(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.receiptPublisher.Publish(ctx, command); err != nil {
		metrics.WSMessagesTotal.Inc(
			h.nodeID,
			"message.receipt",
			"publish_error",
		)

		client.sendError(message.RequestID, "could not accept receipt")
		return
	}

	_ = client.Send(Envelope{
		Type:      TypeReceiptAccepted,
		RequestID: message.RequestID,
		Data: mustJSON(map[string]any{
			"conversation_id": command.ConversationID,
			"sequence":        command.Sequence,
			"receipt_type":    command.Type,
		}),
	})
}

func mustJSON(value any) json.RawMessage {
	payload, _ := json.Marshal(value)
	return payload
}
