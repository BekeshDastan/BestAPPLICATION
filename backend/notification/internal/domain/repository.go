package domain

import (
	"context"

	"github.com/google/uuid"
)

type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*Notification, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Notification, error)
	ListUnreadByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Notification, error)
	MarkAsRead(ctx context.Context, id, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
	DeleteAllRead(ctx context.Context, userID uuid.UUID) error
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
}

type PreferenceRepository interface {
	GetAll(ctx context.Context, userID uuid.UUID) ([]*NotificationPreference, error)
	Upsert(ctx context.Context, p *NotificationPreference) error
}
