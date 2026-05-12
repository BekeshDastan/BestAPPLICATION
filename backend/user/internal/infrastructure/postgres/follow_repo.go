package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/bekesh/social/backend/user/internal/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type FollowRepo struct {
	db *sqlx.DB
}

func NewFollowRepo(db *sqlx.DB) *FollowRepo { return &FollowRepo{db: db} }

func (r *FollowRepo) Follow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	q := querier(ctx, r.db)
	const query = `INSERT INTO follows (follower_id, followee_id, created_at) VALUES ($1, $2, NOW())`
	_, err := q.ExecContext(ctx, query, followerID.String(), followeeID.String())
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return domain.ErrAlreadyFollowing
		}
		return fmt.Errorf("follow: %w", err)
	}
	return nil
}

func (r *FollowRepo) Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	q := querier(ctx, r.db)
	const query = `DELETE FROM follows WHERE follower_id = $1 AND followee_id = $2`
	res, err := q.ExecContext(ctx, query, followerID.String(), followeeID.String())
	if err != nil {
		return fmt.Errorf("unfollow: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFollowing
	}
	return nil
}

func (r *FollowRepo) IsFollowing(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error) {
	q := querier(ctx, r.db)
	var exists bool
	const query = `SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND followee_id = $2)`
	if err := sqlx.GetContext(ctx, q, &exists, query, followerID.String(), followeeID.String()); err != nil {
		return false, fmt.Errorf("is following: %w", err)
	}
	return exists, nil
}

func (r *FollowRepo) ListFollowers(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.User, error) {
	q := querier(ctx, r.db)
	const query = `
		SELECT u.id, u.username, u.email, u.password_hash, u.full_name, u.bio, u.avatar_url,
		       u.is_verified, u.is_private, u.created_at, u.updated_at, u.deleted_at
		FROM users u
		JOIN follows f ON f.follower_id = u.id
		WHERE f.followee_id = $1 AND u.deleted_at IS NULL
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3`
	var rows []userRow
	if err := sqlx.SelectContext(ctx, q, &rows, query, userID.String(), limit, offset); err != nil {
		return nil, fmt.Errorf("list followers: %w", err)
	}
	return rowsToDomain(rows), nil
}

func (r *FollowRepo) ListFollowing(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.User, error) {
	q := querier(ctx, r.db)
	const query = `
		SELECT u.id, u.username, u.email, u.password_hash, u.full_name, u.bio, u.avatar_url,
		       u.is_verified, u.is_private, u.created_at, u.updated_at, u.deleted_at
		FROM users u
		JOIN follows f ON f.followee_id = u.id
		WHERE f.follower_id = $1 AND u.deleted_at IS NULL
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3`
	var rows []userRow
	if err := sqlx.SelectContext(ctx, q, &rows, query, userID.String(), limit, offset); err != nil {
		return nil, fmt.Errorf("list following: %w", err)
	}
	return rowsToDomain(rows), nil
}

func (r *FollowRepo) Block(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	q := querier(ctx, r.db)
	const query = `INSERT INTO blocks (blocker_id, blocked_id, created_at) VALUES ($1, $2, NOW())`
	_, err := q.ExecContext(ctx, query, blockerID.String(), blockedID.String())
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return domain.ErrAlreadyBlocked
		}
		return fmt.Errorf("block: %w", err)
	}
	return nil
}

func (r *FollowRepo) Unblock(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	q := querier(ctx, r.db)
	const query = `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`
	res, err := q.ExecContext(ctx, query, blockerID.String(), blockedID.String())
	if err != nil {
		return fmt.Errorf("unblock: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotBlocked
	}
	return nil
}

func (r *FollowRepo) IsBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error) {
	q := querier(ctx, r.db)
	var exists bool
	const query = `SELECT EXISTS(SELECT 1 FROM blocks WHERE blocker_id = $1 AND blocked_id = $2)`
	if err := sqlx.GetContext(ctx, q, &exists, query, blockerID.String(), blockedID.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("is blocked: %w", err)
	}
	return exists, nil
}

func rowsToDomain(rows []userRow) []*domain.User {
	users := make([]*domain.User, 0, len(rows))
	for _, r := range rows {
		u, err := r.toDomain()
		if err == nil {
			users = append(users, u)
		}
	}
	return users
}
