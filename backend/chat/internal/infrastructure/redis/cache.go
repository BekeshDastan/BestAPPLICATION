package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bekesh/social/backend/chat/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type ChatCache struct {
	rdb *redis.Client
}

func NewChatCache(rdb *redis.Client) *ChatCache {
	return &ChatCache{rdb: rdb}
}

func (c *ChatCache) SetTyping(ctx context.Context, convID, userID uuid.UUID, ttl time.Duration) error {
	key := fmt.Sprintf("chat:typing:%s:%s", convID, userID)
	if err := c.rdb.Set(ctx, key, "1", ttl).Err(); err != nil {
		return fmt.Errorf("set typing: %w", err)
	}
	return nil
}

func (c *ChatCache) IsTyping(ctx context.Context, convID, userID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("chat:typing:%s:%s", convID, userID)
	err := c.rdb.Get(ctx, key).Err()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("is typing: %w", err)
	}
	return true, nil
}

func (c *ChatCache) SetOnline(ctx context.Context, userID uuid.UUID, ttl time.Duration) error {
	key := fmt.Sprintf("chat:online:%s", userID)
	if err := c.rdb.Set(ctx, key, "1", ttl).Err(); err != nil {
		return fmt.Errorf("set online: %w", err)
	}
	return nil
}

func (c *ChatCache) IsOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("chat:online:%s", userID)
	err := c.rdb.Get(ctx, key).Err()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("is online: %w", err)
	}
	return true, nil
}

func (c *ChatCache) IncrUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("notif:unread:%s", userID)
	val, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("incr unread: %w", err)
	}
	c.rdb.Expire(ctx, key, time.Minute)
	return val, nil
}

func (c *ChatCache) GetUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("notif:unread:%s", userID)
	val, err := c.rdb.Get(ctx, key).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, fmt.Errorf("get unread: %w", err)
	}
	return val, nil
}

func (c *ChatCache) DelUnread(ctx context.Context, userID uuid.UUID) error {
	key := fmt.Sprintf("notif:unread:%s", userID)
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("del unread: %w", err)
	}
	return nil
}

// satisfy domain interface check
var _ domain.ChatCache = (*ChatCache)(nil)
