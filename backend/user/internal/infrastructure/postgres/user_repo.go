package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bekesh/social/backend/user/internal/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type UserRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) *UserRepo { return &UserRepo{db: db} }

type userRow struct {
	ID           string         `db:"id"`
	Username     string         `db:"username"`
	Email        string         `db:"email"`
	PasswordHash string         `db:"password_hash"`
	FullName     sql.NullString `db:"full_name"`
	Bio          sql.NullString `db:"bio"`
	AvatarURL    sql.NullString `db:"avatar_url"`
	IsVerified   bool           `db:"is_verified"`
	IsPrivate    bool           `db:"is_private"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
	DeletedAt    *time.Time     `db:"deleted_at"`
}

func (r userRow) toDomain() (*domain.User, error) {
	id, err := uuid.Parse(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parse uuid %s: %w", r.ID, err)
	}
	return &domain.User{
		ID:           id,
		Username:     domain.Username(r.Username),
		Email:        domain.Email(r.Email),
		PasswordHash: r.PasswordHash,
		FullName:     r.FullName.String,
		Bio:          r.Bio.String,
		AvatarURL:    r.AvatarURL.String,
		IsVerified:   r.IsVerified,
		IsPrivate:    r.IsPrivate,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
		DeletedAt:    r.DeletedAt,
	}, nil
}

func (repo *UserRepo) Create(ctx context.Context, u *domain.User) error {
	q := querier(ctx, repo.db)
	const query = `
		INSERT INTO users (id, username, email, password_hash, full_name, bio, avatar_url, is_verified, is_private, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := q.ExecContext(ctx, query,
		u.ID.String(), string(u.Username), string(u.Email), u.PasswordHash,
		nullString(u.FullName), nullString(u.Bio), nullString(u.AvatarURL),
		u.IsVerified, u.IsPrivate, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Constraint {
			case "users_email_key":
				return domain.ErrEmailTaken
			case "users_username_key":
				return domain.ErrUsernameTaken
			}
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (repo *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	q := querier(ctx, repo.db)
	var row userRow
	const query = `SELECT id, username, email, password_hash, full_name, bio, avatar_url, is_verified, is_private, created_at, updated_at, deleted_at FROM users WHERE id = $1`
	if err := sqlx.GetContext(ctx, q, &row, query, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return row.toDomain()
}

func (repo *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	q := querier(ctx, repo.db)
	var row userRow
	const query = `SELECT id, username, email, password_hash, full_name, bio, avatar_url, is_verified, is_private, created_at, updated_at, deleted_at FROM users WHERE email = $1 AND deleted_at IS NULL`
	if err := sqlx.GetContext(ctx, q, &row, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return row.toDomain()
}

func (repo *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	q := querier(ctx, repo.db)
	var row userRow
	const query = `SELECT id, username, email, password_hash, full_name, bio, avatar_url, is_verified, is_private, created_at, updated_at, deleted_at FROM users WHERE username = $1 AND deleted_at IS NULL`
	if err := sqlx.GetContext(ctx, q, &row, query, username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return row.toDomain()
}

func (repo *UserRepo) Update(ctx context.Context, u *domain.User) error {
	q := querier(ctx, repo.db)
	const query = `
		UPDATE users SET username=$1, email=$2, password_hash=$3, full_name=$4, bio=$5, avatar_url=$6,
		is_verified=$7, is_private=$8, updated_at=$9
		WHERE id=$10 AND deleted_at IS NULL`
	res, err := q.ExecContext(ctx, query,
		string(u.Username), string(u.Email), u.PasswordHash,
		nullString(u.FullName), nullString(u.Bio), nullString(u.AvatarURL),
		u.IsVerified, u.IsPrivate, u.UpdatedAt, u.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (repo *UserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	q := querier(ctx, repo.db)
	const query = `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	res, err := q.ExecContext(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (repo *UserRepo) Search(ctx context.Context, query string, limit, offset int) ([]*domain.User, error) {
	q := querier(ctx, repo.db)
	const stmt = `
		SELECT id, username, email, password_hash, full_name, bio, avatar_url, is_verified, is_private, created_at, updated_at, deleted_at
		FROM users
		WHERE deleted_at IS NULL
		  AND (username ILIKE '%' || $1 || '%' OR full_name ILIKE '%' || $1 || '%')
		ORDER BY username
		LIMIT $2 OFFSET $3`
	var rows []userRow
	if err := sqlx.SelectContext(ctx, q, &rows, stmt, query, limit, offset); err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	users := make([]*domain.User, 0, len(rows))
	for _, r := range rows {
		u, err := r.toDomain()
		if err != nil {
			continue
		}
		users = append(users, u)
	}
	return users, nil
}

func (repo *UserRepo) CountFollowers(ctx context.Context, userID uuid.UUID) (int, error) {
	q := querier(ctx, repo.db)
	var count int
	if err := sqlx.GetContext(ctx, q, &count, `SELECT COUNT(*) FROM follows WHERE followee_id = $1`, userID.String()); err != nil {
		return 0, fmt.Errorf("count followers: %w", err)
	}
	return count, nil
}

func (repo *UserRepo) CountFollowing(ctx context.Context, userID uuid.UUID) (int, error) {
	q := querier(ctx, repo.db)
	var count int
	if err := sqlx.GetContext(ctx, q, &count, `SELECT COUNT(*) FROM follows WHERE follower_id = $1`, userID.String()); err != nil {
		return 0, fmt.Errorf("count following: %w", err)
	}
	return count, nil
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
