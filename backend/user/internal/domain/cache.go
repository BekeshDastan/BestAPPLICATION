package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserCache interface {
	GetProfile(ctx context.Context, id uuid.UUID) (*User, error)
	SetProfile(ctx context.Context, u *User, ttl time.Duration) error
	InvalidateProfile(ctx context.Context, id uuid.UUID) error
	IsTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error)
	BlacklistToken(ctx context.Context, tokenHash string, ttl time.Duration) error
}
