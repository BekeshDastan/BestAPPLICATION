package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type StoryCache struct {
	rdb *redis.Client
}

func NewStoryCache(rdb *redis.Client) *StoryCache {
	return &StoryCache{rdb: rdb}
}

func (c *StoryCache) IncrViews(ctx context.Context, storyID uuid.UUID) error {
	key := fmt.Sprintf("story:views:%s", storyID)
	return c.rdb.Incr(ctx, key).Err()
}

func (c *StoryCache) GetViews(ctx context.Context, storyID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("story:views:%s", storyID)
	val, err := c.rdb.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

var _ = time.Second
