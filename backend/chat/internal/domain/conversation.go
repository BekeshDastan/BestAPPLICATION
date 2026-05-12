package domain

import (
	"time"

	"github.com/google/uuid"
)

type ConversationType string

const (
	ConvTypeDirect ConversationType = "direct"
	ConvTypeGroup  ConversationType = "group"
)

type ParticipantRole string

const (
	RoleOwner  ParticipantRole = "owner"
	RoleAdmin  ParticipantRole = "admin"
	RoleMember ParticipantRole = "member"
)

type Conversation struct {
	ID            uuid.UUID
	Type          ConversationType
	Name          string
	AvatarURL     string
	CreatedBy     uuid.UUID
	LastMessageAt *time.Time
	CreatedAt     time.Time
}

type Participant struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Role           ParticipantRole
	JoinedAt       time.Time
	LastReadAt     *time.Time
	UnreadCount    int
}
