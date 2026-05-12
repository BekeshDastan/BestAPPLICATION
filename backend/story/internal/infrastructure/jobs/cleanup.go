package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/bekesh/social/backend/story/internal/domain"
)

type StoryCleanupJob struct {
	stories domain.StoryRepository
}

func NewStoryCleanupJob(stories domain.StoryRepository) *StoryCleanupJob {
	return &StoryCleanupJob{stories: stories}
}

func (j *StoryCleanupJob) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := j.stories.CleanupExpired(ctx); err != nil {
				slog.Error("story cleanup failed", "err", err)
			}
		}
	}
}
