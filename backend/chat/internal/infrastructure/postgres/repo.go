package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bekesh/social/backend/chat/internal/domain"
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

// ── ConversationRepo ───────────────────────────────────────────────────────

type ConversationRepo struct{ db *sqlx.DB }

func NewConversationRepo(db *sqlx.DB) *ConversationRepo { return &ConversationRepo{db: db} }

type convRow struct {
	ID            string         `db:"id"`
	Type          string         `db:"type"`
	Name          sql.NullString `db:"name"`
	AvatarURL     sql.NullString `db:"avatar_url"`
	CreatedBy     string         `db:"created_by"`
	LastMessageAt *time.Time     `db:"last_message_at"`
	CreatedAt     time.Time      `db:"created_at"`
}

func (r convRow) toDomain() (*domain.Conversation, error) {
	id, err := uuid.Parse(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parse conv id: %w", err)
	}
	createdBy, err := uuid.Parse(r.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("parse created_by: %w", err)
	}
	return &domain.Conversation{
		ID:            id,
		Type:          domain.ConversationType(r.Type),
		Name:          r.Name.String,
		AvatarURL:     r.AvatarURL.String,
		CreatedBy:     createdBy,
		LastMessageAt: r.LastMessageAt,
		CreatedAt:     r.CreatedAt,
	}, nil
}

const convCols = `id, type, name, avatar_url, created_by, last_message_at, created_at`

func (repo *ConversationRepo) Create(ctx context.Context, conv *domain.Conversation) error {
	q := querier(ctx, repo.db)
	const stmt = `INSERT INTO conversations (id, type, name, avatar_url, created_by, created_at) VALUES ($1,$2,$3,$4,$5,$6)`
	_, err := q.ExecContext(ctx, stmt,
		conv.ID.String(), string(conv.Type),
		nullString(conv.Name), nullString(conv.AvatarURL),
		conv.CreatedBy.String(), conv.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert conversation: %w", err)
	}
	return nil
}

func (repo *ConversationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error) {
	var row convRow
	stmt := `SELECT ` + convCols + ` FROM conversations WHERE id = $1`
	if err := repo.db.GetContext(ctx, &row, stmt, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrConversationNotFound
		}
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	return row.toDomain()
}

func (repo *ConversationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := repo.db.ExecContext(ctx, `DELETE FROM conversations WHERE id=$1`, id.String())
	if err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrConversationNotFound
	}
	return nil
}

func (repo *ConversationRepo) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Conversation, error) {
	stmt := `SELECT c.` + convCols + `
	FROM conversations c
	JOIN conversation_participants cp ON cp.conversation_id = c.id
	WHERE cp.user_id = $1
	ORDER BY COALESCE(c.last_message_at, c.created_at) DESC
	LIMIT $2 OFFSET $3`

	var rows []convRow
	if err := repo.db.SelectContext(ctx, &rows, stmt, userID.String(), limit, offset); err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	out := make([]*domain.Conversation, 0, len(rows))
	for _, r := range rows {
		c, err := r.toDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (repo *ConversationRepo) UpdateLastMessageAt(ctx context.Context, id uuid.UUID, t time.Time) error {
	q := querier(ctx, repo.db)
	_, err := q.ExecContext(ctx, `UPDATE conversations SET last_message_at=$1 WHERE id=$2`, t, id.String())
	return err
}

func (repo *ConversationRepo) UpdateInfo(ctx context.Context, id uuid.UUID, name, avatarURL string) error {
	_, err := repo.db.ExecContext(ctx,
		`UPDATE conversations SET name=$1, avatar_url=$2 WHERE id=$3`,
		nullString(name), nullString(avatarURL), id.String(),
	)
	return err
}

// ── ParticipantRepo ────────────────────────────────────────────────────────

type ParticipantRepo struct{ db *sqlx.DB }

func NewParticipantRepo(db *sqlx.DB) *ParticipantRepo { return &ParticipantRepo{db: db} }

type participantRow struct {
	ConversationID string     `db:"conversation_id"`
	UserID         string     `db:"user_id"`
	Role           string     `db:"role"`
	JoinedAt       time.Time  `db:"joined_at"`
	LastReadAt     *time.Time `db:"last_read_at"`
	UnreadCount    int        `db:"unread_count"`
}

func (r participantRow) toDomain() (*domain.Participant, error) {
	convID, err := uuid.Parse(r.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("parse conv id: %w", err)
	}
	userID, err := uuid.Parse(r.UserID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}
	return &domain.Participant{
		ConversationID: convID,
		UserID:         userID,
		Role:           domain.ParticipantRole(r.Role),
		JoinedAt:       r.JoinedAt,
		LastReadAt:     r.LastReadAt,
		UnreadCount:    r.UnreadCount,
	}, nil
}

func (repo *ParticipantRepo) Add(ctx context.Context, p *domain.Participant) error {
	q := querier(ctx, repo.db)
	const stmt = `INSERT INTO conversation_participants (conversation_id, user_id, role, joined_at) VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`
	_, err := q.ExecContext(ctx, stmt, p.ConversationID.String(), p.UserID.String(), string(p.Role), p.JoinedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return domain.ErrAlreadyParticipant
		}
		return fmt.Errorf("add participant: %w", err)
	}
	return nil
}

func (repo *ParticipantRepo) Remove(ctx context.Context, convID, userID uuid.UUID) error {
	q := querier(ctx, repo.db)
	res, err := q.ExecContext(ctx, `DELETE FROM conversation_participants WHERE conversation_id=$1 AND user_id=$2`, convID.String(), userID.String())
	if err != nil {
		return fmt.Errorf("remove participant: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotParticipant
	}
	return nil
}

func (repo *ParticipantRepo) IsParticipant(ctx context.Context, convID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := repo.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM conversation_participants WHERE conversation_id=$1 AND user_id=$2)`,
		convID.String(), userID.String(),
	).Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("check participant: %w", err)
	}
	return exists, nil
}

func (repo *ParticipantRepo) ListParticipants(ctx context.Context, convID uuid.UUID) ([]*domain.Participant, error) {
	var rows []participantRow
	const stmt = `SELECT conversation_id, user_id, role, joined_at, last_read_at, unread_count FROM conversation_participants WHERE conversation_id=$1`
	if err := repo.db.SelectContext(ctx, &rows, stmt, convID.String()); err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}
	out := make([]*domain.Participant, 0, len(rows))
	for _, r := range rows {
		p, err := r.toDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (repo *ParticipantRepo) GetParticipant(ctx context.Context, convID, userID uuid.UUID) (*domain.Participant, error) {
	var row participantRow
	const stmt = `SELECT conversation_id, user_id, role, joined_at, last_read_at, unread_count FROM conversation_participants WHERE conversation_id=$1 AND user_id=$2`
	if err := repo.db.GetContext(ctx, &row, stmt, convID.String(), userID.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotParticipant
		}
		return nil, fmt.Errorf("get participant: %w", err)
	}
	return row.toDomain()
}

func (repo *ParticipantRepo) MarkRead(ctx context.Context, convID, userID uuid.UUID) error {
	q := querier(ctx, repo.db)
	_, err := q.ExecContext(ctx,
		`UPDATE conversation_participants SET unread_count=0, last_read_at=NOW() WHERE conversation_id=$1 AND user_id=$2`,
		convID.String(), userID.String(),
	)
	return err
}

func (repo *ParticipantRepo) IncrUnreadExceptSender(ctx context.Context, convID, senderID uuid.UUID) error {
	q := querier(ctx, repo.db)
	_, err := q.ExecContext(ctx,
		`UPDATE conversation_participants SET unread_count=unread_count+1 WHERE conversation_id=$1 AND user_id!=$2`,
		convID.String(), senderID.String(),
	)
	return err
}

// ── MessageRepo ────────────────────────────────────────────────────────────

type MessageRepo struct{ db *sqlx.DB }

func NewMessageRepo(db *sqlx.DB) *MessageRepo { return &MessageRepo{db: db} }

type messageRow struct {
	ID             string         `db:"id"`
	ConversationID string         `db:"conversation_id"`
	SenderID       string         `db:"sender_id"`
	ReplyToID      sql.NullString `db:"reply_to_id"`
	Text           sql.NullString `db:"text"`
	MediaURL       sql.NullString `db:"media_url"`
	IsPinned       bool           `db:"is_pinned"`
	EditedAt       *time.Time     `db:"edited_at"`
	CreatedAt      time.Time      `db:"created_at"`
	DeletedAt      *time.Time     `db:"deleted_at"`
}

func (r messageRow) toDomain() (*domain.Message, error) {
	id, err := uuid.Parse(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parse message id: %w", err)
	}
	convID, err := uuid.Parse(r.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("parse conv id: %w", err)
	}
	senderID, err := uuid.Parse(r.SenderID)
	if err != nil {
		return nil, fmt.Errorf("parse sender id: %w", err)
	}
	m := &domain.Message{
		ID:             id,
		ConversationID: convID,
		SenderID:       senderID,
		Text:           r.Text.String,
		MediaURL:       r.MediaURL.String,
		IsPinned:       r.IsPinned,
		EditedAt:       r.EditedAt,
		CreatedAt:      r.CreatedAt,
		DeletedAt:      r.DeletedAt,
	}
	if r.ReplyToID.Valid {
		rid, err := uuid.Parse(r.ReplyToID.String)
		if err != nil {
			return nil, fmt.Errorf("parse reply_to_id: %w", err)
		}
		m.ReplyToID = &rid
	}
	return m, nil
}

const msgCols = `id, conversation_id, sender_id, reply_to_id, text, media_url, is_pinned, edited_at, created_at, deleted_at`

func (repo *MessageRepo) Create(ctx context.Context, m *domain.Message) error {
	q := querier(ctx, repo.db)
	var replyToID sql.NullString
	if m.ReplyToID != nil {
		replyToID = sql.NullString{String: m.ReplyToID.String(), Valid: true}
	}
	const stmt = `INSERT INTO messages (id, conversation_id, sender_id, reply_to_id, text, media_url, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`
	_, err := q.ExecContext(ctx, stmt,
		m.ID.String(), m.ConversationID.String(), m.SenderID.String(),
		replyToID, nullString(m.Text), nullString(m.MediaURL), m.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	return nil
}

func (repo *MessageRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	var row messageRow
	stmt := `SELECT ` + msgCols + ` FROM messages WHERE id=$1`
	if err := repo.db.GetContext(ctx, &row, stmt, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMessageNotFound
		}
		return nil, fmt.Errorf("get message: %w", err)
	}
	return row.toDomain()
}

func (repo *MessageRepo) Update(ctx context.Context, m *domain.Message) error {
	q := querier(ctx, repo.db)
	const stmt = `UPDATE messages SET text=$1, edited_at=$2 WHERE id=$3 AND deleted_at IS NULL`
	res, err := q.ExecContext(ctx, stmt, nullString(m.Text), m.EditedAt, m.ID.String())
	if err != nil {
		return fmt.Errorf("update message: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrMessageNotFound
	}
	return nil
}

func (repo *MessageRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	q := querier(ctx, repo.db)
	const stmt = `UPDATE messages SET deleted_at=NOW() WHERE id=$1 AND deleted_at IS NULL`
	res, err := q.ExecContext(ctx, stmt, id.String())
	if err != nil {
		return fmt.Errorf("soft delete message: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrMessageNotFound
	}
	return nil
}

func (repo *MessageRepo) ListByConversation(ctx context.Context, convID uuid.UUID, limit, offset int) ([]*domain.Message, error) {
	stmt := `SELECT ` + msgCols + ` FROM messages WHERE conversation_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	var rows []messageRow
	if err := repo.db.SelectContext(ctx, &rows, stmt, convID.String(), limit, offset); err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	return rowsToDomain(rows)
}

func (repo *MessageRepo) Search(ctx context.Context, convID uuid.UUID, query string, limit, offset int) ([]*domain.Message, error) {
	stmt := `SELECT ` + msgCols + ` FROM messages WHERE conversation_id=$1 AND text ILIKE '%' || $2 || '%' AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	var rows []messageRow
	if err := repo.db.SelectContext(ctx, &rows, stmt, convID.String(), query, limit, offset); err != nil {
		return nil, fmt.Errorf("search messages: %w", err)
	}
	return rowsToDomain(rows)
}

func (repo *MessageRepo) SetPinned(ctx context.Context, id uuid.UUID, pinned bool) error {
	_, err := repo.db.ExecContext(ctx, `UPDATE messages SET is_pinned=$1 WHERE id=$2 AND deleted_at IS NULL`, pinned, id.String())
	return err
}

func rowsToDomain(rows []messageRow) ([]*domain.Message, error) {
	out := make([]*domain.Message, 0, len(rows))
	for _, r := range rows {
		m, err := r.toDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// ── ReactionRepo ───────────────────────────────────────────────────────────

type ReactionRepo struct{ db *sqlx.DB }

func NewReactionRepo(db *sqlx.DB) *ReactionRepo { return &ReactionRepo{db: db} }

func (repo *ReactionRepo) Add(ctx context.Context, r *domain.MessageReaction) error {
	const stmt = `INSERT INTO message_reactions (message_id, user_id, emoji) VALUES ($1,$2,$3)`
	_, err := repo.db.ExecContext(ctx, stmt, r.MessageID.String(), r.UserID.String(), r.Emoji)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return domain.ErrDuplicateReaction
		}
		return fmt.Errorf("add reaction: %w", err)
	}
	return nil
}

func (repo *ReactionRepo) Remove(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	res, err := repo.db.ExecContext(ctx,
		`DELETE FROM message_reactions WHERE message_id=$1 AND user_id=$2 AND emoji=$3`,
		messageID.String(), userID.String(), emoji,
	)
	if err != nil {
		return fmt.Errorf("remove reaction: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
