package domain

import (
	"context"

	"github.com/google/uuid"
)

type NotificationCache interface {
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
	SetUnreadCount(ctx context.Context, userID uuid.UUID, count int) error
	IncrUnread(ctx context.Context, userID uuid.UUID) error
	DecrUnread(ctx context.Context, userID uuid.UUID) error
	InvalidateUnread(ctx context.Context, userID uuid.UUID) error
}
