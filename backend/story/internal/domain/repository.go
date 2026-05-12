package domain

import (
	"context"

	"github.com/google/uuid"
)

type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}

type StoryRepository interface {
	Create(ctx context.Context, s *Story) error
	GetByID(ctx context.Context, id uuid.UUID) (*Story, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Story, error)
	ListByUserIDs(ctx context.Context, userIDs []uuid.UUID, limit, offset int) ([]*Story, error)
	IncrViewsCount(ctx context.Context, id uuid.UUID) error
	CleanupExpired(ctx context.Context) error
}

type StoryViewRepository interface {
	Add(ctx context.Context, v *StoryView) error
	IsViewed(ctx context.Context, storyID, viewerID uuid.UUID) (bool, error)
	ListViewers(ctx context.Context, storyID uuid.UUID, limit, offset int) ([]*StoryView, error)
}

type StoryReplyRepository interface {
	Create(ctx context.Context, r *StoryReply) error
	ListByStory(ctx context.Context, storyID uuid.UUID) ([]*StoryReply, error)
}

type StoryReactionRepository interface {
	Add(ctx context.Context, r *StoryReaction) error
	Remove(ctx context.Context, storyID, userID uuid.UUID) error
	GetReactionCounts(ctx context.Context, storyID uuid.UUID) (map[string]int, error)
}

type HighlightRepository interface {
	Create(ctx context.Context, h *Highlight) error
	GetByID(ctx context.Context, id uuid.UUID) (*Highlight, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Highlight, error)
	AddStory(ctx context.Context, hs *HighlightStory) error
	RemoveStory(ctx context.Context, highlightID, storyID uuid.UUID) error
}

// StoryCleanupTarget is satisfied by the cleanup job.
type StoryCleanupTarget interface {
	CleanupExpired(ctx context.Context) error
}


