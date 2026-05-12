package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	MaxCaptionLen = 2200
	MaxMediaCount = 10
)

type Post struct {
	ID            uuid.UUID
	AuthorID      uuid.UUID
	Caption       string
	MediaURLs     []string
	Tags          []string
	LikesCount    int
	CommentsCount int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

func (p *Post) IsDeleted() bool { return p.DeletedAt != nil }
