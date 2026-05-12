package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bekesh/social/backend/post/internal/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// ── Transaction support ────────────────────────────────────────────────────

type txKey struct{}

type DB struct{ db *sqlx.DB }

func NewDB(db *sqlx.DB) *DB { return &DB{db: db} }

func (d *DB) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback after %w: %v", err, rbErr)
		}
		return err
	}
	return tx.Commit()
}

func querier(ctx context.Context, db *sqlx.DB) sqlx.ExtContext {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return db
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// ── PostRepo ───────────────────────────────────────────────────────────────

type PostRepo struct{ db *sqlx.DB }

func NewPostRepo(db *sqlx.DB) *PostRepo { return &PostRepo{db: db} }

type postRow struct {
	ID            string         `db:"id"`
	AuthorID      string         `db:"author_id"`
	Caption       sql.NullString `db:"caption"`
	MediaURLs     pq.StringArray `db:"media_urls"`
	Tags          pq.StringArray `db:"tags"`
	LikesCount    int            `db:"likes_count"`
	CommentsCount int            `db:"comments_count"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
	DeletedAt     *time.Time     `db:"deleted_at"`
}

func (r postRow) toDomain() (*domain.Post, error) {
	id, err := uuid.Parse(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parse post id: %w", err)
	}
	authorID, err := uuid.Parse(r.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("parse author id: %w", err)
	}
	media := []string(r.MediaURLs)
	if media == nil {
		media = []string{}
	}
	tags := []string(r.Tags)
	if tags == nil {
		tags = []string{}
	}
	return &domain.Post{
		ID:            id,
		AuthorID:      authorID,
		Caption:       r.Caption.String,
		MediaURLs:     media,
		Tags:          tags,
		LikesCount:    r.LikesCount,
		CommentsCount: r.CommentsCount,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		DeletedAt:     r.DeletedAt,
	}, nil
}

const postCols = `id, author_id, caption, media_urls, tags, likes_count, comments_count, created_at, updated_at, deleted_at`

func (repo *PostRepo) Create(ctx context.Context, p *domain.Post) error {
	q := querier(ctx, repo.db)
	const stmt = `INSERT INTO posts (id, author_id, caption, media_urls, tags, created_at, updated_at)
	              VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := q.ExecContext(ctx, stmt,
		p.ID.String(), p.AuthorID.String(),
		nullString(p.Caption),
		pq.Array(p.MediaURLs), pq.Array(p.Tags),
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert post: %w", err)
	}
	return nil
}

func (repo *PostRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
	q := querier(ctx, repo.db)
	var row postRow
	stmt := `SELECT ` + postCols + ` FROM posts WHERE id = $1`
	if err := sqlx.GetContext(ctx, q, &row, stmt, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPostNotFound
		}
		return nil, fmt.Errorf("get post: %w", err)
	}
	return row.toDomain()
}

func (repo *PostRepo) Update(ctx context.Context, p *domain.Post) error {
	q := querier(ctx, repo.db)
	const stmt = `UPDATE posts SET caption=$1, tags=$2, updated_at=$3 WHERE id=$4 AND deleted_at IS NULL`
	res, err := q.ExecContext(ctx, stmt, nullString(p.Caption), pq.Array(p.Tags), p.UpdatedAt, p.ID.String())
	if err != nil {
		return fmt.Errorf("update post: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrPostNotFound
	}
	return nil
}

func (repo *PostRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	q := querier(ctx, repo.db)
	const stmt = `UPDATE posts SET deleted_at=NOW() WHERE id=$1 AND deleted_at IS NULL`
	res, err := q.ExecContext(ctx, stmt, id.String())
	if err != nil {
		return fmt.Errorf("soft delete post: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrPostNotFound
	}
	return nil
}

func (repo *PostRepo) ListByAuthor(ctx context.Context, authorID uuid.UUID, limit, offset int) ([]*domain.Post, error) {
	q := querier(ctx, repo.db)
	stmt := `SELECT ` + postCols + ` FROM posts WHERE author_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	var rows []postRow
	if err := sqlx.SelectContext(ctx, q, &rows, stmt, authorID.String(), limit, offset); err != nil {
		return nil, fmt.Errorf("list posts by author: %w", err)
	}
	return rowsToDomain(rows)
}

func (repo *PostRepo) ListByAuthors(ctx context.Context, authorIDs []uuid.UUID, limit, offset int) ([]*domain.Post, error) {
	if len(authorIDs) == 0 {
		return []*domain.Post{}, nil
	}
	ids := make(pq.StringArray, len(authorIDs))
	for i, id := range authorIDs {
		ids[i] = id.String()
	}
	q := querier(ctx, repo.db)
	stmt := `SELECT ` + postCols + ` FROM posts WHERE author_id::text = ANY($1) AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	var rows []postRow
	if err := sqlx.SelectContext(ctx, q, &rows, stmt, ids, limit, offset); err != nil {
		return nil, fmt.Errorf("list feed: %w", err)
	}
	return rowsToDomain(rows)
}

func (repo *PostRepo) Search(ctx context.Context, query string, limit, offset int) ([]*domain.Post, error) {
	q := querier(ctx, repo.db)
	stmt := `SELECT ` + postCols + ` FROM posts WHERE caption ILIKE '%' || $1 || '%' AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	var rows []postRow
	if err := sqlx.SelectContext(ctx, q, &rows, stmt, query, limit, offset); err != nil {
		return nil, fmt.Errorf("search posts: %w", err)
	}
	return rowsToDomain(rows)
}

func (repo *PostRepo) IncrementLikes(ctx context.Context, postID uuid.UUID) error {
	q := querier(ctx, repo.db)
	_, err := q.ExecContext(ctx, `UPDATE posts SET likes_count=likes_count+1 WHERE id=$1`, postID.String())
	return err
}

func (repo *PostRepo) DecrementLikes(ctx context.Context, postID uuid.UUID) error {
	q := querier(ctx, repo.db)
	_, err := q.ExecContext(ctx, `UPDATE posts SET likes_count=GREATEST(0,likes_count-1) WHERE id=$1`, postID.String())
	return err
}

func (repo *PostRepo) IncrementComments(ctx context.Context, postID uuid.UUID) error {
	q := querier(ctx, repo.db)
	_, err := q.ExecContext(ctx, `UPDATE posts SET comments_count=comments_count+1 WHERE id=$1`, postID.String())
	return err
}

func (repo *PostRepo) DecrementComments(ctx context.Context, postID uuid.UUID) error {
	q := querier(ctx, repo.db)
	_, err := q.ExecContext(ctx, `UPDATE posts SET comments_count=GREATEST(0,comments_count-1) WHERE id=$1`, postID.String())
	return err
}

func rowsToDomain(rows []postRow) ([]*domain.Post, error) {
	out := make([]*domain.Post, 0, len(rows))
	for _, r := range rows {
		p, err := r.toDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// ── LikeRepo ───────────────────────────────────────────────────────────────

type LikeRepo struct{ db *sqlx.DB }

func NewLikeRepo(db *sqlx.DB) *LikeRepo { return &LikeRepo{db: db} }

func (repo *LikeRepo) Like(ctx context.Context, postID, userID uuid.UUID) error {
	q := querier(ctx, repo.db)
	_, err := q.ExecContext(ctx, `INSERT INTO likes (post_id, user_id) VALUES ($1, $2)`,
		postID.String(), userID.String())
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return domain.ErrAlreadyLiked
		}
		return fmt.Errorf("like post: %w", err)
	}
	return nil
}

func (repo *LikeRepo) Unlike(ctx context.Context, postID, userID uuid.UUID) error {
	q := querier(ctx, repo.db)
	res, err := q.ExecContext(ctx, `DELETE FROM likes WHERE post_id=$1 AND user_id=$2`,
		postID.String(), userID.String())
	if err != nil {
		return fmt.Errorf("unlike post: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotLiked
	}
	return nil
}

func (repo *LikeRepo) IsLiked(ctx context.Context, postID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := repo.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM likes WHERE post_id=$1 AND user_id=$2)`,
		postID.String(), userID.String()).Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("check like: %w", err)
	}
	return exists, nil
}

func (repo *LikeRepo) ListLikers(ctx context.Context, postID uuid.UUID, limit, offset int) ([]uuid.UUID, error) {
	var ids []string
	if err := repo.db.SelectContext(ctx, &ids,
		`SELECT user_id FROM likes WHERE post_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		postID.String(), limit, offset); err != nil {
		return nil, fmt.Errorf("list likers: %w", err)
	}
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		uid, err := uuid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("parse user id: %w", err)
		}
		result = append(result, uid)
	}
	return result, nil
}

// ── CommentRepo ────────────────────────────────────────────────────────────

type CommentRepo struct{ db *sqlx.DB }

func NewCommentRepo(db *sqlx.DB) *CommentRepo { return &CommentRepo{db: db} }

type commentRow struct {
	ID        string     `db:"id"`
	PostID    string     `db:"post_id"`
	AuthorID  string     `db:"author_id"`
	Body      string     `db:"body"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

func (r commentRow) toDomain() (*domain.Comment, error) {
	id, err := uuid.Parse(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parse comment id: %w", err)
	}
	postID, err := uuid.Parse(r.PostID)
	if err != nil {
		return nil, fmt.Errorf("parse post id: %w", err)
	}
	authorID, err := uuid.Parse(r.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("parse author id: %w", err)
	}
	return &domain.Comment{
		ID:        id,
		PostID:    postID,
		AuthorID:  authorID,
		Body:      r.Body,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		DeletedAt: r.DeletedAt,
	}, nil
}

func (repo *CommentRepo) Create(ctx context.Context, c *domain.Comment) error {
	q := querier(ctx, repo.db)
	const stmt = `INSERT INTO comments (id, post_id, author_id, body, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6)`
	_, err := q.ExecContext(ctx, stmt, c.ID.String(), c.PostID.String(), c.AuthorID.String(), c.Body, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert comment: %w", err)
	}
	return nil
}

func (repo *CommentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	q := querier(ctx, repo.db)
	var row commentRow
	const stmt = `SELECT id, post_id, author_id, body, created_at, updated_at, deleted_at FROM comments WHERE id=$1`
	if err := sqlx.GetContext(ctx, q, &row, stmt, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCommentNotFound
		}
		return nil, fmt.Errorf("get comment: %w", err)
	}
	return row.toDomain()
}

func (repo *CommentRepo) ListByPost(ctx context.Context, postID uuid.UUID, limit, offset int) ([]*domain.Comment, error) {
	var rows []commentRow
	const stmt = `SELECT id, post_id, author_id, body, created_at, updated_at, deleted_at
	              FROM comments WHERE post_id=$1 AND deleted_at IS NULL ORDER BY created_at ASC LIMIT $2 OFFSET $3`
	if err := repo.db.SelectContext(ctx, &rows, stmt, postID.String(), limit, offset); err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	out := make([]*domain.Comment, 0, len(rows))
	for _, r := range rows {
		c, err := r.toDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (repo *CommentRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	q := querier(ctx, repo.db)
	const stmt = `UPDATE comments SET deleted_at=NOW() WHERE id=$1 AND deleted_at IS NULL`
	res, err := q.ExecContext(ctx, stmt, id.String())
	if err != nil {
		return fmt.Errorf("soft delete comment: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrCommentNotFound
	}
	return nil
}
