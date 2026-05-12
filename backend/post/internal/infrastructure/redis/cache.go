package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bekesh/social/backend/post/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type PostCache struct {
	rdb *redis.Client
}

func NewPostCache(rdb *redis.Client) *PostCache {
	return &PostCache{rdb: rdb}
}

func (c *PostCache) GetPost(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
	data, err := c.rdb.Get(ctx, postKey(id)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("redis get post: %w", err)
	}
	var p domain.Post
	if err = json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("unmarshal post: %w", err)
	}
	return &p, nil
}

func (c *PostCache) SetPost(ctx context.Context, p *domain.Post, ttl time.Duration) error {
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal post: %w", err)
	}
	if err = c.rdb.Set(ctx, postKey(p.ID), data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set post: %w", err)
	}
	return nil
}

func (c *PostCache) InvalidatePost(ctx context.Context, id uuid.UUID) error {
	if err := c.rdb.Del(ctx, postKey(id)).Err(); err != nil {
		return fmt.Errorf("redis del post: %w", err)
	}
	return nil
}

func postKey(id uuid.UUID) string { return "post:" + id.String() }
