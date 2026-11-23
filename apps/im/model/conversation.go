package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type ChatType int32

const (
	ChatTypeGroup  ChatType = 1
	ChatTypeSingle ChatType = 2
)

type GroupRole int32

const (
	GroupRoleOwner  GroupRole = 1
	GroupRoleAdmin  GroupRole = 2
	GroupRoleMember GroupRole = 3
)

// Conversation 是共享的会话记录。它不保存每个用户的已读游标或可见性状态。
type Conversation struct {
	ID                string    `bson:"_id"`
	DirectKey         string    `bson:"directKey,omitempty"`
	ChatType          ChatType  `bson:"chatType"`
	LastSeq           int64     `bson:"lastSeq"`
	Dissolved         bool      `bson:"dissolved"`
	MembershipVersion uint64    `bson:"membershipVersion"`
	CreatedAt         time.Time `bson:"createdAt"`
	UpdatedAt         time.Time `bson:"updatedAt"`
}

// ConversationMember 用来存储某一个用户在某一个会话里的状态。
type ConversationMember struct {
	ID             bson.ObjectID `bson:"_id,omitempty"`
	ConversationID string        `bson:"conversationId"`
	UserID         string        `bson:"userId"`

	JoinedSeq    int64 `bson:"joinedSeq"`
	LeftSeq      int64 `bson:"leftSeq"`
	DeliveredSeq int64 `bson:"deliveredSeq"`
	ReadSeq      int64 `bson:"readSeq"`

	IsVisible         bool      `bson:"isVisible"`
	CreatedAt         time.Time `bson:"createdAt"`
	UpdatedAt         time.Time `bson:"updatedAt"`
	LastActivityAt    time.Time `bson:"lastActivityAt"`
	MembershipVersion uint64    `bson:"membershipVersion"`
	GroupRole         int32     `bson:"groupRole"`
}
