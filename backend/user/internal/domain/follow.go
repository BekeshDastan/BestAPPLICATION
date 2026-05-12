package domain

import (
	"time"

	"github.com/google/uuid"
)

type Follow struct {
	FollowerID uuid.UUID
	FolloweeID uuid.UUID
	CreatedAt  time.Time
}

type Block struct {
	BlockerID uuid.UUID
	BlockedID uuid.UUID
	CreatedAt time.Time
}
