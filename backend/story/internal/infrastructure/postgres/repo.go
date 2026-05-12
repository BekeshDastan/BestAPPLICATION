package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bekesh/social/backend/story/internal/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ── Transaction support ────────────────────────────────────────────────────

type txKey struct{}

type DB struct{ db *sqlx.DB }

func NewDB(db *sqlx.DB) *DB { return &DB{db: db} }

func (d *DB) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

type execerCtx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func execer(ctx context.Context, db *sqlx.DB) execerCtx {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return db
}

// ── StoryRepo ──────────────────────────────────────────────────────────────

type StoryRepo struct{ db *sqlx.DB }

func NewStoryRepo(db *sqlx.DB) *StoryRepo { return &StoryRepo{db: db} }

type storyRow struct {
	ID         uuid.UUID  `db:"id"`
	UserID     uuid.UUID  `db:"user_id"`
	MediaURL   string     `db:"media_url"`
	MediaType  string     `db:"media_type"`
	Caption    string     `db:"caption"`
	ExpiresAt  time.Time  `db:"expires_at"`
	ViewsCount int        `db:"views_count"`
	CreatedAt  time.Time  `db:"created_at"`
	DeletedAt  *time.Time `db:"deleted_at"`
}

func toStory(r storyRow) *domain.Story {
	return &domain.Story{
		ID:         r.ID,
		UserID:     r.UserID,
		MediaURL:   r.MediaURL,
		MediaType:  domain.MediaType(r.MediaType),
		Caption:    r.Caption,
		ExpiresAt:  r.ExpiresAt,
		ViewsCount: r.ViewsCount,
		CreatedAt:  r.CreatedAt,
		DeletedAt:  r.DeletedAt,
	}
}

func (r *StoryRepo) Create(ctx context.Context, s *domain.Story) error {
	_, err := execer(ctx, r.db).ExecContext(ctx,
		`INSERT INTO stories (id, user_id, media_url, media_type, caption, expires_at, views_count, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		s.ID, s.UserID, s.MediaURL, string(s.MediaType), s.Caption, s.ExpiresAt, s.ViewsCount, s.CreatedAt)
	return err
}

func (r *StoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Story, error) {
	var row storyRow
	err := sqlx.GetContext(ctx, r.db,
		&row, `SELECT id, user_id, media_url, media_type, caption, expires_at, views_count, created_at, deleted_at
		        FROM stories WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrStoryNotFound
	}
	if err != nil {
		return nil, err
	}
	return toStory(row), nil
}

func (r *StoryRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := execer(ctx, r.db).ExecContext(ctx,
		`UPDATE stories SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

func (r *StoryRepo) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Story, error) {
	var rows []storyRow
	err := sqlx.SelectContext(ctx, r.db, &rows,
		`SELECT id, user_id, media_url, media_type, caption, expires_at, views_count, created_at, deleted_at
		 FROM stories
		 WHERE user_id = $1 AND deleted_at IS NULL AND expires_at > NOW()
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Story, len(rows))
	for i, row := range rows {
		out[i] = toStory(row)
	}
	return out, nil
}

func (r *StoryRepo) ListByUserIDs(ctx context.Context, userIDs []uuid.UUID, limit, offset int) ([]*domain.Story, error) {
	if len(userIDs) == 0 {
		return []*domain.Story{}, nil
	}
	ids := make([]any, len(userIDs))
	for i, id := range userIDs {
		ids[i] = id
	}
	query, args, err := sqlx.In(
		`SELECT id, user_id, media_url, media_type, caption, expires_at, views_count, created_at, deleted_at
		 FROM stories
		 WHERE user_id IN (?) AND deleted_at IS NULL AND expires_at > NOW()
		 ORDER BY created_at DESC
		 LIMIT ? OFFSET ?`,
		ids, limit, offset)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var rows []storyRow
	if err = sqlx.SelectContext(ctx, r.db, &rows, query, args...); err != nil {
		return nil, err
	}
	out := make([]*domain.Story, len(rows))
	for i, row := range rows {
		out[i] = toStory(row)
	}
	return out, nil
}

func (r *StoryRepo) IncrViewsCount(ctx context.Context, id uuid.UUID) error {
	_, err := execer(ctx, r.db).ExecContext(ctx,
		`UPDATE stories SET views_count = views_count + 1 WHERE id = $1`, id)
	return err
}

func (r *StoryRepo) CleanupExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE stories SET deleted_at = NOW() WHERE expires_at < NOW() AND deleted_at IS NULL`)
	return err
}

// ── StoryViewRepo ──────────────────────────────────────────────────────────

type StoryViewRepo struct{ db *sqlx.DB }

func NewStoryViewRepo(db *sqlx.DB) *StoryViewRepo { return &StoryViewRepo{db: db} }

type storyViewRow struct {
	StoryID  uuid.UUID `db:"story_id"`
	ViewerID uuid.UUID `db:"viewer_id"`
	ViewedAt time.Time `db:"viewed_at"`
}

func (r *StoryViewRepo) Add(ctx context.Context, v *domain.StoryView) error {
	_, err := execer(ctx, r.db).ExecContext(ctx,
		`INSERT INTO story_views (story_id, viewer_id, viewed_at) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		v.StoryID, v.ViewerID, v.ViewedAt)
	return err
}

func (r *StoryViewRepo) IsViewed(ctx context.Context, storyID, viewerID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM story_views WHERE story_id = $1 AND viewer_id = $2)`,
		storyID, viewerID).Scan(&exists)
	return exists, err
}

func (r *StoryViewRepo) ListViewers(ctx context.Context, storyID uuid.UUID, limit, offset int) ([]*domain.StoryView, error) {
	var rows []storyViewRow
	err := sqlx.SelectContext(ctx, r.db, &rows,
		`SELECT story_id, viewer_id, viewed_at FROM story_views WHERE story_id = $1 ORDER BY viewed_at DESC LIMIT $2 OFFSET $3`,
		storyID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.StoryView, len(rows))
	for i, row := range rows {
		out[i] = &domain.StoryView{StoryID: row.StoryID, ViewerID: row.ViewerID, ViewedAt: row.ViewedAt}
	}
	return out, nil
}

// ── StoryReplyRepo ─────────────────────────────────────────────────────────

type StoryReplyRepo struct{ db *sqlx.DB }

func NewStoryReplyRepo(db *sqlx.DB) *StoryReplyRepo { return &StoryReplyRepo{db: db} }

type storyReplyRow struct {
	ID        uuid.UUID `db:"id"`
	StoryID   uuid.UUID `db:"story_id"`
	UserID    uuid.UUID `db:"user_id"`
	Text      string    `db:"text"`
	CreatedAt time.Time `db:"created_at"`
}

func (r *StoryReplyRepo) Create(ctx context.Context, reply *domain.StoryReply) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO story_replies (id, story_id, user_id, text, created_at) VALUES ($1, $2, $3, $4, $5)`,
		reply.ID, reply.StoryID, reply.UserID, reply.Text, reply.CreatedAt)
	return err
}

func (r *StoryReplyRepo) ListByStory(ctx context.Context, storyID uuid.UUID) ([]*domain.StoryReply, error) {
	var rows []storyReplyRow
	err := sqlx.SelectContext(ctx, r.db, &rows,
		`SELECT id, story_id, user_id, text, created_at FROM story_replies WHERE story_id = $1 ORDER BY created_at ASC`,
		storyID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.StoryReply, len(rows))
	for i, row := range rows {
		out[i] = &domain.StoryReply{ID: row.ID, StoryID: row.StoryID, UserID: row.UserID, Text: row.Text, CreatedAt: row.CreatedAt}
	}
	return out, nil
}

// ── StoryReactionRepo ──────────────────────────────────────────────────────

type StoryReactionRepo struct{ db *sqlx.DB }

func NewStoryReactionRepo(db *sqlx.DB) *StoryReactionRepo { return &StoryReactionRepo{db: db} }

func (r *StoryReactionRepo) Add(ctx context.Context, react *domain.StoryReaction) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO story_reactions (story_id, user_id, emoji) VALUES ($1, $2, $3)
		 ON CONFLICT (story_id, user_id) DO UPDATE SET emoji = EXCLUDED.emoji`,
		react.StoryID, react.UserID, react.Emoji)
	return err
}

func (r *StoryReactionRepo) Remove(ctx context.Context, storyID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM story_reactions WHERE story_id = $1 AND user_id = $2`, storyID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrReactionNotFound
	}
	return nil
}

func (r *StoryReactionRepo) GetReactionCounts(ctx context.Context, storyID uuid.UUID) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT emoji, COUNT(*) FROM story_reactions WHERE story_id = $1 GROUP BY emoji`, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int)
	for rows.Next() {
		var emoji string
		var count int
		if err := rows.Scan(&emoji, &count); err != nil {
			return nil, err
		}
		counts[emoji] = count
	}
	return counts, rows.Err()
}

// ── HighlightRepo ──────────────────────────────────────────────────────────

type HighlightRepo struct{ db *sqlx.DB }

func NewHighlightRepo(db *sqlx.DB) *HighlightRepo { return &HighlightRepo{db: db} }

type highlightRow struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	Title     string    `db:"title"`
	CoverURL  string    `db:"cover_url"`
	CreatedAt time.Time `db:"created_at"`
}

func (r *HighlightRepo) Create(ctx context.Context, h *domain.Highlight) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO highlights (id, user_id, title, cover_url, created_at) VALUES ($1, $2, $3, $4, $5)`,
		h.ID, h.UserID, h.Title, h.CoverURL, h.CreatedAt)
	return err
}

func (r *HighlightRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Highlight, error) {
	var row highlightRow
	err := sqlx.GetContext(ctx, r.db, &row,
		`SELECT id, user_id, title, COALESCE(cover_url,'') AS cover_url, created_at FROM highlights WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrHighlightNotFound
	}
	if err != nil {
		return nil, err
	}
	return &domain.Highlight{ID: row.ID, UserID: row.UserID, Title: row.Title, CoverURL: row.CoverURL, CreatedAt: row.CreatedAt}, nil
}

func (r *HighlightRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM highlights WHERE id = $1`, id)
	return err
}

func (r *HighlightRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Highlight, error) {
	var rows []highlightRow
	err := sqlx.SelectContext(ctx, r.db, &rows,
		`SELECT id, user_id, title, COALESCE(cover_url,'') AS cover_url, created_at FROM highlights WHERE user_id = $1 ORDER BY created_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Highlight, len(rows))
	for i, row := range rows {
		out[i] = &domain.Highlight{ID: row.ID, UserID: row.UserID, Title: row.Title, CoverURL: row.CoverURL, CreatedAt: row.CreatedAt}
	}
	return out, nil
}

func (r *HighlightRepo) AddStory(ctx context.Context, hs *domain.HighlightStory) error {
	var maxPos sql.NullInt32
	_ = r.db.QueryRowContext(ctx,
		`SELECT MAX(position) FROM highlight_stories WHERE highlight_id = $1`, hs.HighlightID).Scan(&maxPos)
	pos := int32(0)
	if maxPos.Valid {
		pos = maxPos.Int32 + 1
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO highlight_stories (highlight_id, story_id, position) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		hs.HighlightID, hs.StoryID, pos)
	return err
}

func (r *HighlightRepo) RemoveStory(ctx context.Context, highlightID, storyID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM highlight_stories WHERE highlight_id = $1 AND story_id = $2`, highlightID, storyID)
	return err
}

