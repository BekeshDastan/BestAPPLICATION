package domain

import (
	"time"

	"github.com/google/uuid"
)

type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
)

type Story struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	MediaURL   string
	MediaType  MediaType
	Caption    string
	ExpiresAt  time.Time
	ViewsCount int
	CreatedAt  time.Time
	DeletedAt  *time.Time
}

func (s *Story) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

func (s *Story) IsDeleted() bool {
	return s.DeletedAt != nil
}

type StoryView struct {
	StoryID  uuid.UUID
	ViewerID uuid.UUID
	ViewedAt time.Time
}

type StoryReply struct {
	ID        uuid.UUID
	StoryID   uuid.UUID
	UserID    uuid.UUID
	Text      string
	CreatedAt time.Time
}

type StoryReaction struct {
	StoryID uuid.UUID
	UserID  uuid.UUID
	Emoji   string
}

type Highlight struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Title     string
	CoverURL  string
	CreatedAt time.Time
}

type HighlightStory struct {
	HighlightID uuid.UUID
	StoryID     uuid.UUID
	Position    int
}

type StoryAnalytics struct {
	StoryID    uuid.UUID
	ViewsCount int
	Reactions  map[string]int
}
