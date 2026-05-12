package domain

import (
	"time"

	"github.com/google/uuid"
)

const MaxMessageLen = 4000

type Message struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	ReplyToID      *uuid.UUID
	Text           string
	MediaURL       string
	IsPinned       bool
	EditedAt       *time.Time
	CreatedAt      time.Time
	DeletedAt      *time.Time
}

func (m *Message) IsDeleted() bool { return m.DeletedAt != nil }

type MessageReaction struct {
	MessageID uuid.UUID
	UserID    uuid.UUID
	Emoji     string
}
