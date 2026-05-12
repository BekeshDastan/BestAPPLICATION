package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ChatCache interface {
	SetTyping(ctx context.Context, convID, userID uuid.UUID, ttl time.Duration) error
	IsTyping(ctx context.Context, convID, userID uuid.UUID) (bool, error)
	SetOnline(ctx context.Context, userID uuid.UUID, ttl time.Duration) error
	IsOnline(ctx context.Context, userID uuid.UUID) (bool, error)
	IncrUnread(ctx context.Context, userID uuid.UUID) (int64, error)
	GetUnread(ctx context.Context, userID uuid.UUID) (int64, error)
	DelUnread(ctx context.Context, userID uuid.UUID) error
}
