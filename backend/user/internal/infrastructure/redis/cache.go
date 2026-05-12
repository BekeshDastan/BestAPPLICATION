package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bekesh/social/backend/user/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type UserCache struct {
	rdb *redis.Client
}

func NewUserCache(rdb *redis.Client) *UserCache {
	return &UserCache{rdb: rdb}
}

// ── Profile cache ──────────────────────────────────────────────────────────

func (c *UserCache) GetProfile(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	key := profileKey(id)
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("redis get profile: %w", err)
	}
	var u domain.User
	if err = json.Unmarshal(data, &u); err != nil {
		return nil, fmt.Errorf("unmarshal profile: %w", err)
	}
	return &u, nil
}

func (c *UserCache) SetProfile(ctx context.Context, u *domain.User, ttl time.Duration) error {
	data, err := json.Marshal(u)
	if err != nil {
		return fmt.Errorf("marshal profile: %w", err)
	}
	if err = c.rdb.Set(ctx, profileKey(u.ID), data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set profile: %w", err)
	}
	return nil
}

func (c *UserCache) InvalidateProfile(ctx context.Context, id uuid.UUID) error {
	if err := c.rdb.Del(ctx, profileKey(id)).Err(); err != nil {
		return fmt.Errorf("redis del profile: %w", err)
	}
	return nil
}

// ── Token blacklist ────────────────────────────────────────────────────────

func (c *UserCache) IsTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error) {
	exists, err := c.rdb.Exists(ctx, blacklistKey(tokenHash)).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists blacklist: %w", err)
	}
	return exists > 0, nil
}

func (c *UserCache) BlacklistToken(ctx context.Context, tokenHash string, ttl time.Duration) error {
	if err := c.rdb.Set(ctx, blacklistKey(tokenHash), "1", ttl).Err(); err != nil {
		return fmt.Errorf("redis set blacklist: %w", err)
	}
	return nil
}

// ── Keys ───────────────────────────────────────────────────────────────────

func profileKey(id uuid.UUID) string   { return "user:profile:" + id.String() }
func blacklistKey(hash string) string  { return "session:blacklist:" + hash }
