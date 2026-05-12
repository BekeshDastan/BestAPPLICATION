package domain

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string // SHA-256(raw_token), stored in DB
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

func (t *RefreshToken) IsExpired() bool  { return time.Now().After(t.ExpiresAt) }
func (t *RefreshToken) IsRevoked() bool  { return t.RevokedAt != nil }
func (t *RefreshToken) IsValid() bool    { return !t.IsExpired() && !t.IsRevoked() }

type EmailVerification struct {
	Token     string // secure random, URL-safe base64
	UserID    uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (e *EmailVerification) IsExpired() bool { return time.Now().After(e.ExpiresAt) }

type PasswordReset struct {
	Token     string
	UserID    uuid.UUID
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (p *PasswordReset) IsExpired() bool { return time.Now().After(p.ExpiresAt) }
func (p *PasswordReset) IsUsed() bool    { return p.UsedAt != nil }
func (p *PasswordReset) IsValid() bool   { return !p.IsExpired() && !p.IsUsed() }
