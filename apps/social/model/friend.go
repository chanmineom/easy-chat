package model

import "time"

type FriendshipCreatedEvent struct {
	FriendshipID int64     `json:"friendship_id"`
	UserA        string    `json:"user_a"`
	UserB        string    `json:"user_b"`
	Version      uint64    `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
}
