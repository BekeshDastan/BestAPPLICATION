package domain

import (
	"context"

	"github.com/google/uuid"
)

type StoryCache interface {
	IncrViews(ctx context.Context, storyID uuid.UUID) error
	GetViews(ctx context.Context, storyID uuid.UUID) (int64, error)
}
