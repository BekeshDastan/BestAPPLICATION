package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/bekesh/social/backend/story/internal/domain"
	"github.com/bekesh/social/backend/story/internal/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ── Mocks ──────────────────────────────────────────────────────────────────

type mockStoryRepo struct{ mock.Mock }

func (m *mockStoryRepo) Create(ctx context.Context, s *domain.Story) error {
	return m.Called(ctx, s).Error(0)
}
func (m *mockStoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Story, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Story), args.Error(1)
}
func (m *mockStoryRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockStoryRepo) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Story, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]*domain.Story), args.Error(1)
}
func (m *mockStoryRepo) ListByUserIDs(ctx context.Context, userIDs []uuid.UUID, limit, offset int) ([]*domain.Story, error) {
	args := m.Called(ctx, userIDs, limit, offset)
	return args.Get(0).([]*domain.Story), args.Error(1)
}
func (m *mockStoryRepo) IncrViewsCount(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockStoryRepo) CleanupExpired(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

type mockViewRepo struct{ mock.Mock }

func (m *mockViewRepo) Add(ctx context.Context, v *domain.StoryView) error {
	return m.Called(ctx, v).Error(0)
}
func (m *mockViewRepo) IsViewed(ctx context.Context, storyID, viewerID uuid.UUID) (bool, error) {
	args := m.Called(ctx, storyID, viewerID)
	return args.Bool(0), args.Error(1)
}
func (m *mockViewRepo) ListViewers(ctx context.Context, storyID uuid.UUID, limit, offset int) ([]*domain.StoryView, error) {
	args := m.Called(ctx, storyID, limit, offset)
	return args.Get(0).([]*domain.StoryView), args.Error(1)
}

type mockReplyRepo struct{ mock.Mock }

func (m *mockReplyRepo) Create(ctx context.Context, r *domain.StoryReply) error {
	return m.Called(ctx, r).Error(0)
}
func (m *mockReplyRepo) ListByStory(ctx context.Context, storyID uuid.UUID) ([]*domain.StoryReply, error) {
	args := m.Called(ctx, storyID)
	return args.Get(0).([]*domain.StoryReply), args.Error(1)
}

type mockReactionRepo struct{ mock.Mock }

func (m *mockReactionRepo) Add(ctx context.Context, r *domain.StoryReaction) error {
	return m.Called(ctx, r).Error(0)
}
func (m *mockReactionRepo) Remove(ctx context.Context, storyID, userID uuid.UUID) error {
	return m.Called(ctx, storyID, userID).Error(0)
}
func (m *mockReactionRepo) GetReactionCounts(ctx context.Context, storyID uuid.UUID) (map[string]int, error) {
	args := m.Called(ctx, storyID)
	return args.Get(0).(map[string]int), args.Error(1)
}

type mockPublisher struct{ mock.Mock }

func (m *mockPublisher) Publish(ctx context.Context, subject string, payload any) error {
	return m.Called(ctx, subject, payload).Error(0)
}

type mockCache struct{ mock.Mock }

func (m *mockCache) IncrViews(ctx context.Context, storyID uuid.UUID) error {
	return m.Called(ctx, storyID).Error(0)
}
func (m *mockCache) GetViews(ctx context.Context, storyID uuid.UUID) (int64, error) {
	args := m.Called(ctx, storyID)
	return args.Get(0).(int64), args.Error(1)
}

type mockTransactor struct{}

func (m *mockTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func newStoryUC(stories *mockStoryRepo, views *mockViewRepo, replies *mockReplyRepo, reactions *mockReactionRepo) *usecase.StoryUseCase {
	return usecase.NewStoryUseCase(stories, views, replies, reactions, &mockPublisher{}, &mockCache{}, &mockTransactor{})
}

func makeStory(userID uuid.UUID) *domain.Story {
	return &domain.Story{
		ID:        uuid.New(),
		UserID:    userID,
		MediaURL:  "https://cdn.example.com/story.jpg",
		MediaType: domain.MediaTypeImage,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestCreateStory_Success(t *testing.T) {
	stories := &mockStoryRepo{}
	stories.On("Create", mock.Anything, mock.AnythingOfType("*domain.Story")).Return(nil)
	pub := &mockPublisher{}
	pub.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	uc := usecase.NewStoryUseCase(stories, &mockViewRepo{}, &mockReplyRepo{}, &mockReactionRepo{}, pub, &mockCache{}, &mockTransactor{})
	s, err := uc.CreateStory(context.Background(), uuid.New(), "https://cdn.example.com/s.jpg", "image", "caption")

	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Equal(t, domain.MediaTypeImage, s.MediaType)
	stories.AssertExpectations(t)
}

func TestCreateStory_EmptyMediaURL(t *testing.T) {
	uc := newStoryUC(&mockStoryRepo{}, &mockViewRepo{}, &mockReplyRepo{}, &mockReactionRepo{})
	_, err := uc.CreateStory(context.Background(), uuid.New(), "  ", "image", "")
	assert.ErrorIs(t, err, domain.ErrMediaURLRequired)
}

func TestCreateStory_InvalidMediaType(t *testing.T) {
	uc := newStoryUC(&mockStoryRepo{}, &mockViewRepo{}, &mockReplyRepo{}, &mockReactionRepo{})
	_, err := uc.CreateStory(context.Background(), uuid.New(), "https://cdn.example.com/s.mp4", "gif", "")
	assert.ErrorIs(t, err, domain.ErrInvalidMediaType)
}

func TestDeleteStory_NotOwner(t *testing.T) {
	stories := &mockStoryRepo{}
	ownerID := uuid.New()
	s := makeStory(ownerID)
	stories.On("GetByID", mock.Anything, s.ID).Return(s, nil)

	uc := newStoryUC(stories, &mockViewRepo{}, &mockReplyRepo{}, &mockReactionRepo{})
	err := uc.DeleteStory(context.Background(), s.ID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestDeleteStory_Success(t *testing.T) {
	stories := &mockStoryRepo{}
	ownerID := uuid.New()
	s := makeStory(ownerID)
	stories.On("GetByID", mock.Anything, s.ID).Return(s, nil)
	stories.On("SoftDelete", mock.Anything, s.ID).Return(nil)

	uc := newStoryUC(stories, &mockViewRepo{}, &mockReplyRepo{}, &mockReactionRepo{})
	err := uc.DeleteStory(context.Background(), s.ID, ownerID)
	require.NoError(t, err)
	stories.AssertExpectations(t)
}

func TestMarkStoryViewed_AlreadyViewed(t *testing.T) {
	stories := &mockStoryRepo{}
	views := &mockViewRepo{}

	userID := uuid.New()
	s := makeStory(uuid.New())
	stories.On("GetByID", mock.Anything, s.ID).Return(s, nil)
	views.On("IsViewed", mock.Anything, s.ID, userID).Return(true, nil)

	uc := newStoryUC(stories, views, &mockReplyRepo{}, &mockReactionRepo{})
	err := uc.MarkStoryViewed(context.Background(), s.ID, userID)
	assert.ErrorIs(t, err, domain.ErrAlreadyViewed)
}

func TestMarkStoryViewed_Expired(t *testing.T) {
	stories := &mockStoryRepo{}

	s := makeStory(uuid.New())
	s.ExpiresAt = time.Now().Add(-1 * time.Hour)
	stories.On("GetByID", mock.Anything, s.ID).Return(s, nil)

	uc := newStoryUC(stories, &mockViewRepo{}, &mockReplyRepo{}, &mockReactionRepo{})
	err := uc.MarkStoryViewed(context.Background(), s.ID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrStoryExpired)
}

func TestMarkStoryViewed_Success(t *testing.T) {
	stories := &mockStoryRepo{}
	views := &mockViewRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	viewerID := uuid.New()
	s := makeStory(uuid.New())
	stories.On("GetByID", mock.Anything, s.ID).Return(s, nil)
	views.On("IsViewed", mock.Anything, s.ID, viewerID).Return(false, nil)
	views.On("Add", mock.Anything, mock.AnythingOfType("*domain.StoryView")).Return(nil)
	stories.On("IncrViewsCount", mock.Anything, s.ID).Return(nil)
	cache.On("IncrViews", mock.Anything, s.ID).Return(nil)
	pub.On("Publish", mock.Anything, domain.EventStoryViewed, mock.Anything).Return(nil)

	uc := usecase.NewStoryUseCase(stories, views, &mockReplyRepo{}, &mockReactionRepo{}, pub, cache, &mockTransactor{})
	err := uc.MarkStoryViewed(context.Background(), s.ID, viewerID)
	require.NoError(t, err)
	stories.AssertExpectations(t)
	views.AssertExpectations(t)
}

func TestListStoryViewers_NotOwner(t *testing.T) {
	stories := &mockStoryRepo{}
	ownerID := uuid.New()
	s := makeStory(ownerID)
	stories.On("GetByID", mock.Anything, s.ID).Return(s, nil)

	uc := newStoryUC(stories, &mockViewRepo{}, &mockReplyRepo{}, &mockReactionRepo{})
	_, err := uc.ListStoryViewers(context.Background(), s.ID, uuid.New(), 20, 0)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestReplyToStory_EmptyText(t *testing.T) {
	uc := newStoryUC(&mockStoryRepo{}, &mockViewRepo{}, &mockReplyRepo{}, &mockReactionRepo{})
	s := makeStory(uuid.New())
	stories := &mockStoryRepo{}
	stories.On("GetByID", mock.Anything, s.ID).Return(s, nil)
	uc2 := newStoryUC(stories, &mockViewRepo{}, &mockReplyRepo{}, &mockReactionRepo{})
	_, err := uc2.ReplyToStory(context.Background(), s.ID, uuid.New(), "  ")
	assert.ErrorIs(t, err, domain.ErrReplyTextEmpty)
	_ = uc
}

func TestReplyToStory_Success(t *testing.T) {
	stories := &mockStoryRepo{}
	replies := &mockReplyRepo{}

	userID := uuid.New()
	s := makeStory(uuid.New())
	stories.On("GetByID", mock.Anything, s.ID).Return(s, nil)
	replies.On("Create", mock.Anything, mock.AnythingOfType("*domain.StoryReply")).Return(nil)

	uc := newStoryUC(stories, &mockViewRepo{}, replies, &mockReactionRepo{})
	r, err := uc.ReplyToStory(context.Background(), s.ID, userID, "love this!")
	require.NoError(t, err)
	assert.Equal(t, "love this!", r.Text)
}

func TestListFollowingStories_Empty(t *testing.T) {
	uc := newStoryUC(&mockStoryRepo{}, &mockViewRepo{}, &mockReplyRepo{}, &mockReactionRepo{})
	stories, err := uc.ListFollowingStories(context.Background(), []uuid.UUID{}, 20, 0)
	require.NoError(t, err)
	assert.Len(t, stories, 0)
}

func TestGetStoryAnalytics_NotOwner(t *testing.T) {
	stories := &mockStoryRepo{}
	ownerID := uuid.New()
	s := makeStory(ownerID)
	stories.On("GetByID", mock.Anything, s.ID).Return(s, nil)

	uc := newStoryUC(stories, &mockViewRepo{}, &mockReplyRepo{}, &mockReactionRepo{})
	_, err := uc.GetStoryAnalytics(context.Background(), s.ID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestGetStoryAnalytics_Success(t *testing.T) {
	stories := &mockStoryRepo{}
	reactions := &mockReactionRepo{}

	ownerID := uuid.New()
	s := makeStory(ownerID)
	s.ViewsCount = 42
	stories.On("GetByID", mock.Anything, s.ID).Return(s, nil)
	reactions.On("GetReactionCounts", mock.Anything, s.ID).Return(map[string]int{"❤️": 5, "🔥": 3}, nil)

	uc := newStoryUC(stories, &mockViewRepo{}, &mockReplyRepo{}, reactions)
	analytics, err := uc.GetStoryAnalytics(context.Background(), s.ID, ownerID)
	require.NoError(t, err)
	assert.Equal(t, 42, analytics.ViewsCount)
	assert.Equal(t, 5, analytics.Reactions["❤️"])
}
