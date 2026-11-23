package session

import "context"

type Session struct {
	SessionID   string `json:"session_id"`
	UserID      string `json:"user_id"`
	DeviceID    string `json:"device_id"`
	NodeID      string `json:"node_id"`
	RPCAddr     string `json:"rpc_addr"`
	ConnectedAt int64  `json:"connected_at"`
}

type Registry interface {
	Register(ctx context.Context, session Session) error
	Touch(ctx context.Context, session Session) (bool, error)
	Unregister(ctx context.Context, userID, sessionID string) error
	Sessions(ctx context.Context, userID string) ([]Session, error)
	TTL() int
}
