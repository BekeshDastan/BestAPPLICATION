package unit_test

import (
	"context"
	"testing"

	"github.com/bekesh/social/backend/post/internal/domain"
	"github.com/bekesh/social/backend/post/internal/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ── Mocks ──────────────────────────────────────────────────────────────────

type mockLikeRepo struct{ mock.Mock }

func (m *mockLikeRepo) Like(ctx context.Context, postID, userID uuid.UUID) error {
	return m.Called(ctx, postID, userID).Error(0)
}
func (m *mockLikeRepo) Unlike(ctx context.Context, postID, userID uuid.UUID) error {
	return m.Called(ctx, postID, userID).Error(0)
}
func (m *mockLikeRepo) IsLiked(ctx context.Context, postID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, postID, userID)
	return args.Bool(0), args.Error(1)
}
func (m *mockLikeRepo) ListLikers(ctx context.Context, postID uuid.UUID, limit, offset int) ([]uuid.UUID, error) {
	args := m.Called(ctx, postID, limit, offset)
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

type mockTransactor struct{}

func (m *mockTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func newLikeUC(postRepo *mockPostRepo, likeRepo *mockLikeRepo, cache *mockPostCache, pub *mockPublisher) *usecase.LikeUseCase {
	return usecase.NewLikeUseCase(postRepo, likeRepo, cache, pub, &mockTransactor{})
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestLikePost_Success(t *testing.T) {
	postRepo := &mockPostRepo{}
	likeRepo := &mockLikeRepo{}
	cache := &mockPostCache{}
	pub := &mockPublisher{}

	p := makePost()
	postID, userID := p.ID, uuid.New()

	postRepo.On("GetByID", mock.Anything, postID).Return(p, nil)
	likeRepo.On("IsLiked", mock.Anything, postID, userID).Return(false, nil)
	likeRepo.On("Like", mock.Anything, postID, userID).Return(nil)
	postRepo.On("IncrementLikes", mock.Anything, postID).Return(nil)
	cache.On("InvalidatePost", mock.Anything, postID).Return(nil)
	pub.On("Publish", mock.Anything, domain.EventPostLiked, mock.Anything).Return(nil)

	uc := newLikeUC(postRepo, likeRepo, cache, pub)
	err := uc.LikePost(context.Background(), postID, userID)
	require.NoError(t, err)
	postRepo.AssertExpectations(t)
	likeRepo.AssertExpectations(t)
}

func TestLikePost_AlreadyLiked(t *testing.T) {
	postRepo := &mockPostRepo{}
	likeRepo := &mockLikeRepo{}

	p := makePost()
	postID, userID := p.ID, uuid.New()

	postRepo.On("GetByID", mock.Anything, postID).Return(p, nil)
	likeRepo.On("IsLiked", mock.Anything, postID, userID).Return(true, nil)

	uc := newLikeUC(postRepo, likeRepo, &mockPostCache{}, &mockPublisher{})
	err := uc.LikePost(context.Background(), postID, userID)
	assert.ErrorIs(t, err, domain.ErrAlreadyLiked)
}

func TestLikePost_PostNotFound(t *testing.T) {
	postRepo := &mockPostRepo{}
	likeRepo := &mockLikeRepo{}

	postID := uuid.New()
	postRepo.On("GetByID", mock.Anything, postID).Return(nil, domain.ErrPostNotFound)

	uc := newLikeUC(postRepo, likeRepo, &mockPostCache{}, &mockPublisher{})
	err := uc.LikePost(context.Background(), postID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrPostNotFound)
}

func TestUnlikePost_Success(t *testing.T) {
	postRepo := &mockPostRepo{}
	likeRepo := &mockLikeRepo{}
	cache := &mockPostCache{}
	pub := &mockPublisher{}

	p := makePost()
	postID, userID := p.ID, uuid.New()

	postRepo.On("GetByID", mock.Anything, postID).Return(p, nil)
	likeRepo.On("IsLiked", mock.Anything, postID, userID).Return(true, nil)
	likeRepo.On("Unlike", mock.Anything, postID, userID).Return(nil)
	postRepo.On("DecrementLikes", mock.Anything, postID).Return(nil)
	cache.On("InvalidatePost", mock.Anything, postID).Return(nil)
	pub.On("Publish", mock.Anything, domain.EventPostUnliked, mock.Anything).Return(nil)

	uc := newLikeUC(postRepo, likeRepo, cache, pub)
	err := uc.UnlikePost(context.Background(), postID, userID)
	require.NoError(t, err)
}

func TestUnlikePost_NotLiked(t *testing.T) {
	postRepo := &mockPostRepo{}
	likeRepo := &mockLikeRepo{}

	p := makePost()
	postID, userID := p.ID, uuid.New()

	postRepo.On("GetByID", mock.Anything, postID).Return(p, nil)
	likeRepo.On("IsLiked", mock.Anything, postID, userID).Return(false, nil)

	uc := newLikeUC(postRepo, likeRepo, &mockPostCache{}, &mockPublisher{})
	err := uc.UnlikePost(context.Background(), postID, userID)
	assert.ErrorIs(t, err, domain.ErrNotLiked)
}

func TestIsLiked(t *testing.T) {
	postRepo := &mockPostRepo{}
	likeRepo := &mockLikeRepo{}

	postID, userID := uuid.New(), uuid.New()
	likeRepo.On("IsLiked", mock.Anything, postID, userID).Return(true, nil)

	uc := newLikeUC(postRepo, likeRepo, &mockPostCache{}, &mockPublisher{})
	ok, err := uc.IsLiked(context.Background(), postID, userID)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestListLikers_Success(t *testing.T) {
	postRepo := &mockPostRepo{}
	likeRepo := &mockLikeRepo{}

	postID := uuid.New()
	expected := []uuid.UUID{uuid.New(), uuid.New()}
	likeRepo.On("ListLikers", mock.Anything, postID, 20, 0).Return(expected, nil)

	uc := newLikeUC(postRepo, likeRepo, &mockPostCache{}, &mockPublisher{})
	got, err := uc.ListLikers(context.Background(), postID, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestListLikers_ClampsLimit(t *testing.T) {
	postRepo := &mockPostRepo{}
	likeRepo := &mockLikeRepo{}

	postID := uuid.New()
	likeRepo.On("ListLikers", mock.Anything, postID, 20, 0).Return([]uuid.UUID{}, nil)

	uc := newLikeUC(postRepo, likeRepo, &mockPostCache{}, &mockPublisher{})
	_, err := uc.ListLikers(context.Background(), postID, 999, 0)
	require.NoError(t, err)
	likeRepo.AssertExpectations(t) // verifies limit was clamped to 20
}
