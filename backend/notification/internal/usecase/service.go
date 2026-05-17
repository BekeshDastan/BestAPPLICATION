package usecase

import (
	"context"
	"strings"

	"github.com/bekesh/social/backend/notification/internal/domain"
	"github.com/google/uuid"
)

func clamp(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// ── NotificationUseCase ───────────────────────────────────────────────────

type NotificationUseCase struct {
	repo  domain.NotificationRepository
	cache domain.NotificationCache
	pub   domain.EventPublisher
}

func NewNotificationUseCase(
	repo domain.NotificationRepository,
	cache domain.NotificationCache,
	pub domain.EventPublisher,
) *NotificationUseCase {
	return &NotificationUseCase{repo: repo, cache: cache, pub: pub}
}

func (uc *NotificationUseCase) Create(ctx context.Context, n *domain.Notification) (*domain.Notification, error) {
	if n.UserID == uuid.Nil {
		return nil, domain.ErrUserIDRequired
	}
	if strings.TrimSpace(n.Message) == "" {
		return nil, domain.ErrMessageRequired
	}
	n.ID = uuid.New()

	if err := uc.repo.Create(ctx, n); err != nil {
		return nil, err
	}
	_ = uc.cache.IncrUnread(ctx, n.UserID)
	_ = uc.pub.Publish(ctx, domain.EventNotificationCreated, map[string]string{
		"notification_id": n.ID.String(),
		"user_id":         n.UserID.String(),
		"type":            string(n.Type),
	})
	return n, nil
}

func (uc *NotificationUseCase) GetByID(ctx context.Context, id, callerID uuid.UUID) (*domain.Notification, error) {
	n, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if n.UserID != callerID {
		return nil, domain.ErrForbidden
	}
	return n, nil
}

func (uc *NotificationUseCase) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Notification, error) {
	limit, offset = clamp(limit, offset)
	return uc.repo.ListByUser(ctx, userID, limit, offset)
}

func (uc *NotificationUseCase) ListUnread(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Notification, error) {
	limit, offset = clamp(limit, offset)
	return uc.repo.ListUnreadByUser(ctx, userID, limit, offset)
}

func (uc *NotificationUseCase) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	n, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if n.UserID != userID {
		return domain.ErrForbidden
	}
	if err := uc.repo.MarkAsRead(ctx, id, userID); err != nil {
		return err
	}
	_ = uc.cache.DecrUnread(ctx, userID)
	return nil
}

func (uc *NotificationUseCase) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	if err := uc.repo.MarkAllAsRead(ctx, userID); err != nil {
		return err
	}
	_ = uc.cache.InvalidateUnread(ctx, userID)
	return nil
}

func (uc *NotificationUseCase) Delete(ctx context.Context, id, userID uuid.UUID) error {
	n, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if n.UserID != userID {
		return domain.ErrForbidden
	}
	if err := uc.repo.Delete(ctx, id, userID); err != nil {
		return err
	}
	if !n.IsRead {
		_ = uc.cache.DecrUnread(ctx, userID)
	}
	return nil
}

func (uc *NotificationUseCase) DeleteAllRead(ctx context.Context, userID uuid.UUID) error {
	return uc.repo.DeleteAllRead(ctx, userID)
}

func (uc *NotificationUseCase) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := uc.cache.GetUnreadCount(ctx, userID)
	if err == nil {
		return count, nil
	}
	// cache miss — fall through to DB
	n, err := uc.repo.CountUnread(ctx, userID)
	if err != nil {
		return 0, err
	}
	_ = uc.cache.SetUnreadCount(ctx, userID, n)
	return int64(n), nil
}

// ── PreferenceUseCase ─────────────────────────────────────────────────────

type PreferenceUseCase struct {
	repo domain.PreferenceRepository
}

func NewPreferenceUseCase(repo domain.PreferenceRepository) *PreferenceUseCase {
	return &PreferenceUseCase{repo: repo}
}

func (uc *PreferenceUseCase) GetPreferences(ctx context.Context, userID uuid.UUID) ([]*domain.NotificationPreference, error) {
	return uc.repo.GetAll(ctx, userID)
}

func (uc *PreferenceUseCase) UpdatePreference(ctx context.Context, p *domain.NotificationPreference) error {
	if p.UserID == uuid.Nil {
		return domain.ErrUserIDRequired
	}
	return uc.repo.Upsert(ctx, p)
}
