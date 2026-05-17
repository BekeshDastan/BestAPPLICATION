package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bekesh/social/backend/notification/internal/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ── NotificationRepo ──────────────────────────────────────────────────────

type NotificationRepo struct{ db *sqlx.DB }

func NewNotificationRepo(db *sqlx.DB) *NotificationRepo { return &NotificationRepo{db: db} }

type notifRow struct {
	ID            uuid.UUID `db:"id"`
	UserID        uuid.UUID `db:"user_id"`
	ActorID       uuid.UUID `db:"actor_id"`
	Type          string    `db:"type"`
	ReferenceID   uuid.UUID `db:"reference_id"`
	ReferenceType string    `db:"reference_type"`
	Message       string    `db:"message"`
	IsRead        bool      `db:"is_read"`
	CreatedAt     time.Time `db:"created_at"`
}

func toNotification(r notifRow) *domain.Notification {
	return &domain.Notification{
		ID:            r.ID,
		UserID:        r.UserID,
		ActorID:       r.ActorID,
		Type:          domain.NotificationType(r.Type),
		ReferenceID:   r.ReferenceID,
		ReferenceType: r.ReferenceType,
		Message:       r.Message,
		IsRead:        r.IsRead,
		CreatedAt:     r.CreatedAt,
	}
}

func (r *NotificationRepo) Create(ctx context.Context, n *domain.Notification) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, actor_id, type, reference_id, reference_type, message, is_read, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		n.ID, n.UserID, n.ActorID, string(n.Type),
		n.ReferenceID, n.ReferenceType, n.Message, n.IsRead, n.CreatedAt,
	)
	return err
}

func (r *NotificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	var row notifRow
	err := r.db.GetContext(ctx, &row, `
		SELECT id, user_id, actor_id, type, reference_id, reference_type, message, is_read, created_at
		FROM notifications WHERE id=$1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotificationNotFound
		}
		return nil, err
	}
	return toNotification(row), nil
}

func (r *NotificationRepo) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Notification, error) {
	var rows []notifRow
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, user_id, actor_id, type, reference_id, reference_type, message, is_read, created_at
		FROM notifications WHERE user_id=$1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Notification, len(rows))
	for i, row := range rows {
		out[i] = toNotification(row)
	}
	return out, nil
}

func (r *NotificationRepo) ListUnreadByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Notification, error) {
	var rows []notifRow
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, user_id, actor_id, type, reference_id, reference_type, message, is_read, created_at
		FROM notifications WHERE user_id=$1 AND is_read=false
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Notification, len(rows))
	for i, row := range rows {
		out[i] = toNotification(row)
	}
	return out, nil
}

func (r *NotificationRepo) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET is_read=true WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *NotificationRepo) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET is_read=true WHERE user_id=$1 AND is_read=false`, userID)
	return err
}

func (r *NotificationRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM notifications WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *NotificationRepo) DeleteAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM notifications WHERE user_id=$1 AND is_read=true`, userID)
	return err
}

func (r *NotificationRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM notifications WHERE user_id=$1 AND is_read=false`, userID)
	return count, err
}

// ── PreferenceRepo ─────────────────────────────────────────────────────────

type PreferenceRepo struct{ db *sqlx.DB }

func NewPreferenceRepo(db *sqlx.DB) *PreferenceRepo { return &PreferenceRepo{db: db} }

type prefRow struct {
	UserID       uuid.UUID `db:"user_id"`
	Type         string    `db:"type"`
	EmailEnabled bool      `db:"email_enabled"`
	PushEnabled  bool      `db:"push_enabled"`
}

func (r *PreferenceRepo) GetAll(ctx context.Context, userID uuid.UUID) ([]*domain.NotificationPreference, error) {
	var rows []prefRow
	err := r.db.SelectContext(ctx, &rows,
		`SELECT user_id, type, email_enabled, push_enabled
		 FROM notification_preferences WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.NotificationPreference, len(rows))
	for i, row := range rows {
		out[i] = &domain.NotificationPreference{
			UserID:       row.UserID,
			Type:         domain.NotificationType(row.Type),
			EmailEnabled: row.EmailEnabled,
			PushEnabled:  row.PushEnabled,
		}
	}
	return out, nil
}

func (r *PreferenceRepo) Upsert(ctx context.Context, p *domain.NotificationPreference) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notification_preferences (user_id, type, email_enabled, push_enabled)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (user_id, type) DO UPDATE
		SET email_enabled=EXCLUDED.email_enabled, push_enabled=EXCLUDED.push_enabled`,
		p.UserID, string(p.Type), p.EmailEnabled, p.PushEnabled,
	)
	return err
}
