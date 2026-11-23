package gateway

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/metrics"
	"github.com/chanmineom/easy-chat/apps/im/model"
)

const (
	maxIdentifierBytes = 128
	maxContentBytes    = 32 * 1024
)

type SendMessageData struct {
	ConversationID  string `json:"conversation_id"`
	ClientMessageID string `json:"client_message_id"`
	ContentType     string `json:"content_type"`
	Content         string `json:"content"`
}

func (h *ConnectHandler) handleSendMessage(client *Client, envelop Envelope) {
	if envelop.RequestID == "" {
		client.sendError("", "request_id is required")
		return
	}

	var data SendMessageData
	if err := json.Unmarshal(envelop.Data, &data); err != nil {
		client.sendError(envelop.RequestID, "invalid message data")
		return
	}

	if messageDataInvalid(data) {
		client.sendError(envelop.RequestID, "invalid message data")
		return
	}

	command := model.SendMessageCommand{
		CommandID:       newSessionID(),
		ConversationID:  strings.TrimSpace(data.ConversationID),
		SenderID:        client.UserID,
		ClientMessageID: strings.TrimSpace(data.ClientMessageID),
		ContentType:     strings.TrimSpace(data.ContentType),
		Content:         data.Content,
		RequestedAt:     time.Now().UnixMilli(),
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := h.chatPublisher.Publish(ctx, command); err != nil {
		metrics.WSMessagesTotal.Inc(
			h.nodeID,
			"message.send",
			"publish_error",
		)
		client.sendError(envelop.RequestID, "message could not be accepted")
		return
	}

	metrics.WSMessagesTotal.Inc(
		h.nodeID,
		"message.Send",
		"accepted",
	)

	_ = client.Send(Envelope{
		Type:      TypeMessageAccepted,
		RequestID: envelop.RequestID,
		Data: mustJSON(map[string]any{
			"command_id":        command.CommandID,
			"conversation_id":   command.ConversationID,
			"client_message_id": command.ClientMessageID,
		}),
	})
}

func messageDataInvalid(data SendMessageData) bool {
	conversationID := strings.TrimSpace(data.ConversationID)
	clientMessageID := strings.TrimSpace(data.ClientMessageID)
	contentType := strings.TrimSpace(data.ContentType)

	if conversationID == "" || len(conversationID) > maxIdentifierBytes {
		return true
	}
	if clientMessageID == "" || len(clientMessageID) > maxIdentifierBytes {
		return true
	}
	if contentType != model.ContentTypeText {
		return true
	}
	if strings.TrimSpace(data.Content) == "" ||
		len(data.Content) > maxContentBytes {
		return true
	}

	return false
}
