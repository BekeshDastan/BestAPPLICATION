package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type ConversationRepository interface {
	Create(ctx context.Context, conv *Conversation) error
	GetByID(ctx context.Context, id uuid.UUID) (*Conversation, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Conversation, error)
	UpdateLastMessageAt(ctx context.Context, id uuid.UUID, t time.Time) error
	UpdateInfo(ctx context.Context, id uuid.UUID, name, avatarURL string) error
}

type ParticipantRepository interface {
	Add(ctx context.Context, p *Participant) error
	Remove(ctx context.Context, convID, userID uuid.UUID) error
	IsParticipant(ctx context.Context, convID, userID uuid.UUID) (bool, error)
	ListParticipants(ctx context.Context, convID uuid.UUID) ([]*Participant, error)
	GetParticipant(ctx context.Context, convID, userID uuid.UUID) (*Participant, error)
	MarkRead(ctx context.Context, convID, userID uuid.UUID) error
	IncrUnreadExceptSender(ctx context.Context, convID, senderID uuid.UUID) error
}

type MessageRepository interface {
	Create(ctx context.Context, m *Message) error
	GetByID(ctx context.Context, id uuid.UUID) (*Message, error)
	Update(ctx context.Context, m *Message) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	ListByConversation(ctx context.Context, convID uuid.UUID, limit, offset int) ([]*Message, error)
	Search(ctx context.Context, convID uuid.UUID, query string, limit, offset int) ([]*Message, error)
	SetPinned(ctx context.Context, id uuid.UUID, pinned bool) error
}

type ReactionRepository interface {
	Add(ctx context.Context, r *MessageReaction) error
	Remove(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
}
