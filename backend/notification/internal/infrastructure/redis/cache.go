package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bekesh/social/backend/notification/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const unreadTTL = 24 * time.Hour

type NotificationCache struct{ rdb *redis.Client }

func NewNotificationCache(rdb *redis.Client) *NotificationCache {
	return &NotificationCache{rdb: rdb}
}

func unreadKey(userID uuid.UUID) string {
	return fmt.Sprintf("notif:unread:%s", userID)
}

func (c *NotificationCache) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	val, err := c.rdb.Get(ctx, unreadKey(userID)).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, domain.ErrNotificationNotFound
	}
	return val, err
}

func (c *NotificationCache) SetUnreadCount(ctx context.Context, userID uuid.UUID, count int) error {
	return c.rdb.Set(ctx, unreadKey(userID), count, unreadTTL).Err()
}

func (c *NotificationCache) IncrUnread(ctx context.Context, userID uuid.UUID) error {
	pipe := c.rdb.Pipeline()
	pipe.Incr(ctx, unreadKey(userID))
	pipe.Expire(ctx, unreadKey(userID), unreadTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *NotificationCache) DecrUnread(ctx context.Context, userID uuid.UUID) error {
	key := unreadKey(userID)
	val, err := c.rdb.Get(ctx, key).Int64()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}
	if val > 0 {
		pipe := c.rdb.Pipeline()
		pipe.Decr(ctx, key)
		pipe.Expire(ctx, key, unreadTTL)
		_, err = pipe.Exec(ctx)
	}
	return err
}

func (c *NotificationCache) InvalidateUnread(ctx context.Context, userID uuid.UUID) error {
	return c.rdb.Del(ctx, unreadKey(userID)).Err()
}
