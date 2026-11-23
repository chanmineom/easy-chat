package model

type SendMessageCommand struct {
	CommandID       string `json:"command_id"`
	ConversationID  string `json:"conversation_id"`
	SenderID        string `json:"sender_id"`
	ClientMessageID string `json:"client_message_id"`
	ContentType     string `json:"content_type"`
	Content         string `json:"content"`
	RequestedAt     int64  `json:"requested_at"`
}
