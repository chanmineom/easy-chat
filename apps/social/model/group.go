package model

import "time"

type GroupRole int8

const (
	GroupRoleOwner  GroupRole = 1
	GroupRoleAdmin  GroupRole = 2
	GroupRoleMember GroupRole = 3
)

type GroupMemberStatus int8

const (
	GroupMemberActive    GroupMemberStatus = 1
	GroupMemberLeft      GroupMemberStatus = 2
	GroupMemberKicked    GroupMemberStatus = 3
	GroupMemberDissolved GroupMemberStatus = 4
)

type GroupMemberSnapshot struct {
	UserID   string            `json:"user_id"`
	Role     GroupRole         `json:"role"`
	Status   GroupMemberStatus `json:"status"`
	JoinedAt time.Time         `json:"joined_at"`
}

type GroupCreatedEvent struct {
	GroupID           string                `json:"group_id"`
	Name              string                `json:"name"`
	AvatarURL         string                `json:"avatar_url"`
	Notification      string                `json:"notification"`
	OwnerID           string                `json:"owner_id"`
	MembershipVersion int64                 `json:"membership_version"`
	Members           []GroupMemberSnapshot `json:"members"`
	CreatedAt         time.Time             `json:"created_at"`
}

type GroupMemberLeftEvent struct {
	GroupID           string    `json:"group_id"`
	UserID            string    `json:"user_id"`
	ActorID           string    `json:"actor_id"` // 表示执行者
	MembershipVersion uint64    `json:"membership_version"`
	LeftAt            time.Time `json:"left_at"`
}

type GroupMembersAddedEvent struct {
	GroupID           string    `json:"group_id"`
	ActorID           string    `json:"actor_id"`
	MembershipVersion uint64    `json:"membership_version"`
	UserIDs           []string  `json:"user_ids"`
	JoinedAt          time.Time `json:"joined_at"`
}

type GroupMemberKickedEvent struct {
	GroupID           string    `json:"group_id"`
	ActorID           string    `json:"actor_id"`
	UserID            string    `json:"user_id"`
	MembershipVersion uint64    `json:"membership_version"`
	KickedAt          time.Time `json:"kicked_at"`
}

type GroupRoleChangedEvent struct {
	GroupID           string    `json:"group_id"`
	ActorID           string    `json:"actor_id"`
	UserID            string    `json:"user_id"`
	OldRole           GroupRole `json:"old_role"`
	NewRole           GroupRole `json:"new_role"`
	MembershipVersion uint64    `json:"membership_version"`
	ChangedAt         time.Time `json:"changed_at"`
}

type GroupOwnerTransferredEvent struct {
	GroupID           string    `json:"group_id"`
	PreviousOwnerID   string    `json:"previous_owner_id"`
	NewOwnerID        string    `json:"new_owner_id"`
	MembershipVersion uint64    `json:"membership_version"`
	TransferredAt     time.Time `json:"transferred_at"`
}

type GroupDissolveEvent struct {
	GroupID           string    `json:"group_id"`
	ActorID             string    `json:"actor_id"`
	MembershipVersion uint64    `json:"membership_version"`
	DissolvedAt       time.Time `json:"dissolved_at"`
}
