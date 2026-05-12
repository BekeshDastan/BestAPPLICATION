package domain

import (
	"context"

	"github.com/google/uuid"
)

// Transactor runs fn inside a single DB transaction.
// All repository calls made within fn share the same tx via context.
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, u *User) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Search(ctx context.Context, query string, limit, offset int) ([]*User, error)
	CountFollowers(ctx context.Context, userID uuid.UUID) (int, error)
	CountFollowing(ctx context.Context, userID uuid.UUID) (int, error)
}

type FollowRepository interface {
	Follow(ctx context.Context, followerID, followeeID uuid.UUID) error
	Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error
	IsFollowing(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error)
	ListFollowers(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*User, error)
	ListFollowing(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*User, error)
	Block(ctx context.Context, blockerID, blockedID uuid.UUID) error
	Unblock(ctx context.Context, blockerID, blockedID uuid.UUID) error
	IsBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error)
}

type TokenRepository interface {
	SaveRefreshToken(ctx context.Context, t *RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error
	SaveEmailVerification(ctx context.Context, ev *EmailVerification) error
	GetEmailVerification(ctx context.Context, token string) (*EmailVerification, error)
	DeleteEmailVerification(ctx context.Context, token string) error
	SavePasswordReset(ctx context.Context, pr *PasswordReset) error
	GetPasswordReset(ctx context.Context, token string) (*PasswordReset, error)
	MarkPasswordResetUsed(ctx context.Context, token string) error
}
