package domain

import (
	"time"

	"github.com/google/uuid"
)

const MaxCommentLen = 1000

type Comment struct {
	ID        uuid.UUID
	PostID    uuid.UUID
	AuthorID  uuid.UUID
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func (c *Comment) IsDeleted() bool { return c.DeletedAt != nil }

type Like struct {
	PostID    uuid.UUID
	UserID    uuid.UUID
	CreatedAt time.Time
}
