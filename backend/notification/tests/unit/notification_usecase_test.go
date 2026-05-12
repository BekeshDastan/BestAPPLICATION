package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/bekesh/social/backend/notification/internal/domain"
	"github.com/bekesh/social/backend/notification/internal/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ── Mocks ──────────────────────────────────────────────────────────────────

type mockNotifRepo struct{ mock.Mock }

func (m *mockNotifRepo) Create(ctx context.Context, n *domain.Notification) error {
	return m.Called(ctx, n).Error(0)
}
func (m *mockNotifRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}
func (m *mockNotifRepo) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Notification, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]*domain.Notification), args.Error(1)
}
func (m *mockNotifRepo) ListUnreadByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Notification, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]*domain.Notification), args.Error(1)
}
func (m *mockNotifRepo) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	return m.Called(ctx, id, userID).Error(0)
}
func (m *mockNotifRepo) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}
func (m *mockNotifRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return m.Called(ctx, id, userID).Error(0)
}
func (m *mockNotifRepo) DeleteAllRead(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}
func (m *mockNotifRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

type mockCache struct{ mock.Mock }

func (m *mockCache) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return int64(args.Int(0)), args.Error(1)
}
func (m *mockCache) SetUnreadCount(ctx context.Context, userID uuid.UUID, count int) error {
	return m.Called(ctx, userID, count).Error(0)
}
func (m *mockCache) IncrUnread(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}
func (m *mockCache) DecrUnread(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}
func (m *mockCache) InvalidateUnread(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

type mockPublisher struct{ mock.Mock }

func (m *mockPublisher) Publish(ctx context.Context, subject string, payload any) error {
	return m.Called(ctx, subject, payload).Error(0)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func newUC(repo *mockNotifRepo, cache *mockCache, pub *mockPublisher) *usecase.NotificationUseCase {
	return usecase.NewNotificationUseCase(repo, cache, pub)
}

func sampleNotif(userID uuid.UUID) *domain.Notification {
	return &domain.Notification{
		UserID:        userID,
		ActorID:       uuid.New(),
		Type:          domain.NotificationTypeLike,
		ReferenceID:   uuid.New(),
		ReferenceType: "post",
		Message:       "liked your post",
		CreatedAt:     time.Now(),
	}
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestCreate_Success(t *testing.T) {
	repo := &mockNotifRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	userID := uuid.New()
	n := sampleNotif(userID)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)
	cache.On("IncrUnread", mock.Anything, userID).Return(nil)
	pub.On("Publish", mock.Anything, domain.EventNotificationCreated, mock.Anything).Return(nil)

	uc := newUC(repo, cache, pub)
	created, err := uc.Create(context.Background(), n)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, created.ID)
	repo.AssertExpectations(t)
}

func TestCreate_EmptyMessage(t *testing.T) {
	repo := &mockNotifRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	n := sampleNotif(uuid.New())
	n.Message = ""

	uc := newUC(repo, cache, pub)
	_, err := uc.Create(context.Background(), n)

	assert.ErrorIs(t, err, domain.ErrMessageRequired)
}

func TestCreate_EmptyUserID(t *testing.T) {
	repo := &mockNotifRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	n := sampleNotif(uuid.Nil)

	uc := newUC(repo, cache, pub)
	_, err := uc.Create(context.Background(), n)

	assert.ErrorIs(t, err, domain.ErrUserIDRequired)
}

func TestGetByID_Success(t *testing.T) {
	repo := &mockNotifRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	userID := uuid.New()
	notifID := uuid.New()
	n := &domain.Notification{ID: notifID, UserID: userID, Message: "test", CreatedAt: time.Now()}

	repo.On("GetByID", mock.Anything, notifID).Return(n, nil)

	uc := newUC(repo, cache, pub)
	got, err := uc.GetByID(context.Background(), notifID, userID)

	require.NoError(t, err)
	assert.Equal(t, notifID, got.ID)
}

func TestGetByID_Forbidden(t *testing.T) {
	repo := &mockNotifRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	ownerID := uuid.New()
	callerID := uuid.New()
	notifID := uuid.New()
	n := &domain.Notification{ID: notifID, UserID: ownerID}

	repo.On("GetByID", mock.Anything, notifID).Return(n, nil)

	uc := newUC(repo, cache, pub)
	_, err := uc.GetByID(context.Background(), notifID, callerID)

	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestMarkAsRead_Success(t *testing.T) {
	repo := &mockNotifRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	userID := uuid.New()
	notifID := uuid.New()
	n := &domain.Notification{ID: notifID, UserID: userID, IsRead: false}

	repo.On("GetByID", mock.Anything, notifID).Return(n, nil)
	repo.On("MarkAsRead", mock.Anything, notifID, userID).Return(nil)
	cache.On("DecrUnread", mock.Anything, userID).Return(nil)

	uc := newUC(repo, cache, pub)
	err := uc.MarkAsRead(context.Background(), notifID, userID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestMarkAllAsRead_Success(t *testing.T) {
	repo := &mockNotifRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	userID := uuid.New()

	repo.On("MarkAllAsRead", mock.Anything, userID).Return(nil)
	cache.On("InvalidateUnread", mock.Anything, userID).Return(nil)

	uc := newUC(repo, cache, pub)
	err := uc.MarkAllAsRead(context.Background(), userID)

	require.NoError(t, err)
}

func TestDelete_Success(t *testing.T) {
	repo := &mockNotifRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	userID := uuid.New()
	notifID := uuid.New()
	n := &domain.Notification{ID: notifID, UserID: userID, IsRead: false}

	repo.On("GetByID", mock.Anything, notifID).Return(n, nil)
	repo.On("Delete", mock.Anything, notifID, userID).Return(nil)
	cache.On("DecrUnread", mock.Anything, userID).Return(nil)

	uc := newUC(repo, cache, pub)
	err := uc.Delete(context.Background(), notifID, userID)

	require.NoError(t, err)
}

func TestDelete_Forbidden(t *testing.T) {
	repo := &mockNotifRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	ownerID := uuid.New()
	callerID := uuid.New()
	notifID := uuid.New()
	n := &domain.Notification{ID: notifID, UserID: ownerID}

	repo.On("GetByID", mock.Anything, notifID).Return(n, nil)

	uc := newUC(repo, cache, pub)
	err := uc.Delete(context.Background(), notifID, callerID)

	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestGetUnreadCount_CacheHit(t *testing.T) {
	repo := &mockNotifRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	userID := uuid.New()
	cache.On("GetUnreadCount", mock.Anything, userID).Return(5, nil)

	uc := newUC(repo, cache, pub)
	count, err := uc.GetUnreadCount(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
	repo.AssertNotCalled(t, "CountUnread")
}

func TestGetUnreadCount_CacheMiss(t *testing.T) {
	repo := &mockNotifRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	userID := uuid.New()
	cache.On("GetUnreadCount", mock.Anything, userID).Return(0, domain.ErrNotificationNotFound)
	repo.On("CountUnread", mock.Anything, userID).Return(3, nil)
	cache.On("SetUnreadCount", mock.Anything, userID, 3).Return(nil)

	uc := newUC(repo, cache, pub)
	count, err := uc.GetUnreadCount(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestListByUser_Clamps(t *testing.T) {
	repo := &mockNotifRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	userID := uuid.New()
	repo.On("ListByUser", mock.Anything, userID, 20, 0).Return([]*domain.Notification{}, nil)

	uc := newUC(repo, cache, pub)
	_, err := uc.ListByUser(context.Background(), userID, -1, -5)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}
