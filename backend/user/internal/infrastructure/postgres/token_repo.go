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
)

type TokenRepo struct {
	db *sqlx.DB
}

func NewTokenRepo(db *sqlx.DB) *TokenRepo { return &TokenRepo{db: db} }

// ── RefreshToken ───────────────────────────────────────────────────────────

func (r *TokenRepo) SaveRefreshToken(ctx context.Context, t *domain.RefreshToken) error {
	q := querier(ctx, r.db)
	const query = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := q.ExecContext(ctx, query,
		t.ID.String(), t.UserID.String(), t.TokenHash, t.ExpiresAt, t.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

func (r *TokenRepo) GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	q := querier(ctx, r.db)
	type row struct {
		ID        string     `db:"id"`
		UserID    string     `db:"user_id"`
		TokenHash string     `db:"token_hash"`
		ExpiresAt time.Time  `db:"expires_at"`
		RevokedAt *time.Time `db:"revoked_at"`
		CreatedAt time.Time  `db:"created_at"`
	}
	var rt row
	const query = `SELECT id, user_id, token_hash, expires_at, revoked_at, created_at FROM refresh_tokens WHERE token_hash = $1`
	if err := sqlx.GetContext(ctx, q, &rt, query, tokenHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	id, _ := uuid.Parse(rt.ID)
	userID, _ := uuid.Parse(rt.UserID)
	return &domain.RefreshToken{
		ID:        id,
		UserID:    userID,
		TokenHash: rt.TokenHash,
		ExpiresAt: rt.ExpiresAt,
		RevokedAt: rt.RevokedAt,
		CreatedAt: rt.CreatedAt,
	}, nil
}

func (r *TokenRepo) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	q := querier(ctx, r.db)
	const query = `UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`
	_, err := q.ExecContext(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *TokenRepo) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	q := querier(ctx, r.db)
	const query = `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := q.ExecContext(ctx, query, userID.String())
	if err != nil {
		return fmt.Errorf("revoke all user tokens: %w", err)
	}
	return nil
}

// ── EmailVerification ──────────────────────────────────────────────────────

func (r *TokenRepo) SaveEmailVerification(ctx context.Context, ev *domain.EmailVerification) error {
	q := querier(ctx, r.db)
	const query = `
		INSERT INTO email_verifications (token, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (token) DO NOTHING`
	_, err := q.ExecContext(ctx, query, ev.Token, ev.UserID.String(), ev.ExpiresAt, ev.CreatedAt)
	if err != nil {
		return fmt.Errorf("save email verification: %w", err)
	}
	return nil
}

func (r *TokenRepo) GetEmailVerification(ctx context.Context, token string) (*domain.EmailVerification, error) {
	q := querier(ctx, r.db)
	type row struct {
		Token     string    `db:"token"`
		UserID    string    `db:"user_id"`
		ExpiresAt time.Time `db:"expires_at"`
		CreatedAt time.Time `db:"created_at"`
	}
	var ev row
	const query = `SELECT token, user_id, expires_at, created_at FROM email_verifications WHERE token = $1`
	if err := sqlx.GetContext(ctx, q, &ev, query, token); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get email verification: %w", err)
	}
	userID, _ := uuid.Parse(ev.UserID)
	return &domain.EmailVerification{
		Token:     ev.Token,
		UserID:    userID,
		ExpiresAt: ev.ExpiresAt,
		CreatedAt: ev.CreatedAt,
	}, nil
}

func (r *TokenRepo) DeleteEmailVerification(ctx context.Context, token string) error {
	q := querier(ctx, r.db)
	_, err := q.ExecContext(ctx, `DELETE FROM email_verifications WHERE token = $1`, token)
	if err != nil {
		return fmt.Errorf("delete email verification: %w", err)
	}
	return nil
}

// ── PasswordReset ──────────────────────────────────────────────────────────

func (r *TokenRepo) SavePasswordReset(ctx context.Context, pr *domain.PasswordReset) error {
	q := querier(ctx, r.db)
	const query = `
		INSERT INTO password_resets (token, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4)`
	_, err := q.ExecContext(ctx, query, pr.Token, pr.UserID.String(), pr.ExpiresAt, pr.CreatedAt)
	if err != nil {
		return fmt.Errorf("save password reset: %w", err)
	}
	return nil
}

func (r *TokenRepo) GetPasswordReset(ctx context.Context, token string) (*domain.PasswordReset, error) {
	q := querier(ctx, r.db)
	type row struct {
		Token     string     `db:"token"`
		UserID    string     `db:"user_id"`
		ExpiresAt time.Time  `db:"expires_at"`
		UsedAt    *time.Time `db:"used_at"`
		CreatedAt time.Time  `db:"created_at"`
	}
	var pr row
	const query = `SELECT token, user_id, expires_at, used_at, created_at FROM password_resets WHERE token = $1`
	if err := sqlx.GetContext(ctx, q, &pr, query, token); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get password reset: %w", err)
	}
	userID, _ := uuid.Parse(pr.UserID)
	return &domain.PasswordReset{
		Token:     pr.Token,
		UserID:    userID,
		ExpiresAt: pr.ExpiresAt,
		UsedAt:    pr.UsedAt,
		CreatedAt: pr.CreatedAt,
	}, nil
}

func (r *TokenRepo) MarkPasswordResetUsed(ctx context.Context, token string) error {
	q := querier(ctx, r.db)
	const query = `UPDATE password_resets SET used_at = NOW() WHERE token = $1 AND used_at IS NULL`
	_, err := q.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("mark password reset used: %w", err)
	}
	return nil
}
