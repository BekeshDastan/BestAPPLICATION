package domain

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ── Value Objects ──────────────────────────────────────────────────────────

type Email string

var emailRx = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func NewEmail(s string) (Email, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if !emailRx.MatchString(s) {
		return "", ErrInvalidEmail
	}
	return Email(s), nil
}

type Username string

var usernameRx = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

func NewUsername(s string) (Username, error) {
	s = strings.TrimSpace(s)
	if !usernameRx.MatchString(s) {
		return "", ErrInvalidUsername
	}
	return Username(s), nil
}

// RawPassword is a plain-text password before hashing — validated on creation.
type RawPassword string

func NewRawPassword(s string) (RawPassword, error) {
	if len(s) < 8 {
		return "", ErrWeakPassword
	}
	return RawPassword(s), nil
}

// ── Entity ─────────────────────────────────────────────────────────────────

type User struct {
	ID           uuid.UUID
	Username     Username
	Email        Email
	PasswordHash string
	FullName     string
	Bio          string
	AvatarURL    string
	IsVerified   bool
	IsPrivate    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}
