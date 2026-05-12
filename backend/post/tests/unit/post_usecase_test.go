package unit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bekesh/social/backend/post/internal/domain"
	"github.com/bekesh/social/backend/post/internal/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ── Mocks ──────────────────────────────────────────────────────────────────

type mockPostRepo struct{ mock.Mock }

func (m *mockPostRepo) Create(ctx context.Context, p *domain.Post) error {
	return m.Called(ctx, p).Error(0)
}
func (m *mockPostRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Post), args.Error(1)
}
func (m *mockPostRepo) Update(ctx context.Context, p *domain.Post) error {
	return m.Called(ctx, p).Error(0)
}
func (m *mockPostRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockPostRepo) ListByAuthor(ctx context.Context, authorID uuid.UUID, limit, offset int) ([]*domain.Post, error) {
	args := m.Called(ctx, authorID, limit, offset)
	return args.Get(0).([]*domain.Post), args.Error(1)
}
func (m *mockPostRepo) ListByAuthors(ctx context.Context, ids []uuid.UUID, limit, offset int) ([]*domain.Post, error) {
	args := m.Called(ctx, ids, limit, offset)
	return args.Get(0).([]*domain.Post), args.Error(1)
}
func (m *mockPostRepo) Search(ctx context.Context, query string, limit, offset int) ([]*domain.Post, error) {
	args := m.Called(ctx, query, limit, offset)
	return args.Get(0).([]*domain.Post), args.Error(1)
}
func (m *mockPostRepo) IncrementLikes(ctx context.Context, postID uuid.UUID) error {
	return m.Called(ctx, postID).Error(0)
}
func (m *mockPostRepo) DecrementLikes(ctx context.Context, postID uuid.UUID) error {
	return m.Called(ctx, postID).Error(0)
}
func (m *mockPostRepo) IncrementComments(ctx context.Context, postID uuid.UUID) error {
	return m.Called(ctx, postID).Error(0)
}
func (m *mockPostRepo) DecrementComments(ctx context.Context, postID uuid.UUID) error {
	return m.Called(ctx, postID).Error(0)
}

type mockPostCache struct{ mock.Mock }

func (m *mockPostCache) GetPost(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Post), args.Error(1)
}
func (m *mockPostCache) SetPost(ctx context.Context, p *domain.Post, ttl time.Duration) error {
	return m.Called(ctx, p, ttl).Error(0)
}
func (m *mockPostCache) InvalidatePost(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

type mockPublisher struct{ mock.Mock }

func (m *mockPublisher) Publish(ctx context.Context, subject string, payload any) error {
	return m.Called(ctx, subject, payload).Error(0)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func newPostUC(repo *mockPostRepo, cache *mockPostCache, pub *mockPublisher) *usecase.PostUseCase {
	return usecase.NewPostUseCase(repo, cache, pub)
}

func makePost(opts ...func(*domain.Post)) *domain.Post {
	p := &domain.Post{
		ID:        uuid.New(),
		AuthorID:  uuid.New(),
		Caption:   "test caption",
		MediaURLs: []string{"https://example.com/img.jpg"},
		Tags:      []string{"go", "test"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestCreatePost_Success(t *testing.T) {
	repo := &mockPostRepo{}
	cache := &mockPostCache{}
	pub := &mockPublisher{}

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Post")).Return(nil)
	pub.On("Publish", mock.Anything, domain.EventPostCreated, mock.Anything).Return(nil)

	uc := newPostUC(repo, cache, pub)
	p, err := uc.CreatePost(context.Background(), usecase.CreatePostInput{
		AuthorID:  uuid.New(),
		Caption:   "hello world",
		MediaURLs: []string{"https://example.com/img.jpg"},
		Tags:      []string{"#go", "test"},
	})

	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, []string{"go", "test"}, p.Tags) // tags normalized
	repo.AssertExpectations(t)
}

func TestCreatePost_CaptionTooLong(t *testing.T) {
	uc := newPostUC(&mockPostRepo{}, &mockPostCache{}, &mockPublisher{})
	long := make([]byte, domain.MaxCaptionLen+1)
	_, err := uc.CreatePost(context.Background(), usecase.CreatePostInput{
		AuthorID:  uuid.New(),
		Caption:   string(long),
		MediaURLs: []string{"https://example.com/img.jpg"},
	})
	assert.ErrorIs(t, err, domain.ErrCaptionTooLong)
}

func TestCreatePost_EmptyMedia(t *testing.T) {
	uc := newPostUC(&mockPostRepo{}, &mockPostCache{}, &mockPublisher{})
	_, err := uc.CreatePost(context.Background(), usecase.CreatePostInput{
		AuthorID:  uuid.New(),
		Caption:   "caption",
		MediaURLs: []string{},
	})
	assert.ErrorIs(t, err, domain.ErrEmptyMedia)
}

func TestCreatePost_TooManyMedia(t *testing.T) {
	uc := newPostUC(&mockPostRepo{}, &mockPostCache{}, &mockPublisher{})
	urls := make([]string, domain.MaxMediaCount+1)
	for i := range urls {
		urls[i] = "https://example.com/img.jpg"
	}
	_, err := uc.CreatePost(context.Background(), usecase.CreatePostInput{
		AuthorID:  uuid.New(),
		Caption:   "caption",
		MediaURLs: urls,
	})
	assert.ErrorIs(t, err, domain.ErrTooManyMedia)
}

func TestGetPost_CacheHit(t *testing.T) {
	repo := &mockPostRepo{}
	cache := &mockPostCache{}
	pub := &mockPublisher{}

	p := makePost()
	cache.On("GetPost", mock.Anything, p.ID).Return(p, nil)

	uc := newPostUC(repo, cache, pub)
	got, err := uc.GetPost(context.Background(), p.ID)

	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)
	repo.AssertNotCalled(t, "GetByID")
}

func TestGetPost_CacheMiss(t *testing.T) {
	repo := &mockPostRepo{}
	cache := &mockPostCache{}
	pub := &mockPublisher{}

	p := makePost()
	cache.On("GetPost", mock.Anything, p.ID).Return(nil, domain.ErrNotFound)
	repo.On("GetByID", mock.Anything, p.ID).Return(p, nil)
	cache.On("SetPost", mock.Anything, p, mock.AnythingOfType("time.Duration")).Return(nil)

	uc := newPostUC(repo, cache, pub)
	got, err := uc.GetPost(context.Background(), p.ID)

	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)
	repo.AssertExpectations(t)
}

func TestGetPost_NotFound(t *testing.T) {
	repo := &mockPostRepo{}
	cache := &mockPostCache{}

	cache.On("GetPost", mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, domain.ErrPostNotFound)

	uc := newPostUC(repo, cache, &mockPublisher{})
	_, err := uc.GetPost(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrPostNotFound)
}

func TestGetPost_DeletedPost(t *testing.T) {
	repo := &mockPostRepo{}
	cache := &mockPostCache{}

	now := time.Now()
	p := makePost(func(p *domain.Post) { p.DeletedAt = &now })

	cache.On("GetPost", mock.Anything, p.ID).Return(nil, domain.ErrNotFound)
	repo.On("GetByID", mock.Anything, p.ID).Return(p, nil)

	uc := newPostUC(repo, cache, &mockPublisher{})
	_, err := uc.GetPost(context.Background(), p.ID)
	assert.ErrorIs(t, err, domain.ErrPostNotFound)
}

func TestUpdatePost_Success(t *testing.T) {
	repo := &mockPostRepo{}
	cache := &mockPostCache{}
	pub := &mockPublisher{}

	p := makePost()
	repo.On("GetByID", mock.Anything, p.ID).Return(p, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Post")).Return(nil)
	cache.On("InvalidatePost", mock.Anything, p.ID).Return(nil)

	uc := newPostUC(repo, cache, pub)
	updated, err := uc.UpdatePost(context.Background(), p.ID, p.AuthorID, usecase.UpdatePostInput{Caption: "new caption", Tags: []string{"new"}})

	require.NoError(t, err)
	assert.Equal(t, "new caption", updated.Caption)
}

func TestUpdatePost_NotOwner(t *testing.T) {
	repo := &mockPostRepo{}
	cache := &mockPostCache{}

	p := makePost()
	repo.On("GetByID", mock.Anything, p.ID).Return(p, nil)

	uc := newPostUC(repo, cache, &mockPublisher{})
	_, err := uc.UpdatePost(context.Background(), p.ID, uuid.New(), usecase.UpdatePostInput{Caption: "x"})
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestDeletePost_Success(t *testing.T) {
	repo := &mockPostRepo{}
	cache := &mockPostCache{}
	pub := &mockPublisher{}

	p := makePost()
	repo.On("GetByID", mock.Anything, p.ID).Return(p, nil)
	repo.On("SoftDelete", mock.Anything, p.ID).Return(nil)
	cache.On("InvalidatePost", mock.Anything, p.ID).Return(nil)
	pub.On("Publish", mock.Anything, domain.EventPostDeleted, mock.Anything).Return(nil)

	uc := newPostUC(repo, cache, pub)
	err := uc.DeletePost(context.Background(), p.ID, p.AuthorID)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeletePost_NotOwner(t *testing.T) {
	repo := &mockPostRepo{}
	p := makePost()
	repo.On("GetByID", mock.Anything, p.ID).Return(p, nil)

	uc := newPostUC(repo, &mockPostCache{}, &mockPublisher{})
	err := uc.DeletePost(context.Background(), p.ID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestListUserPosts_Success(t *testing.T) {
	repo := &mockPostRepo{}
	posts := []*domain.Post{makePost(), makePost()}
	repo.On("ListByAuthor", mock.Anything, mock.Anything, 20, 0).Return(posts, nil)

	uc := newPostUC(repo, &mockPostCache{}, &mockPublisher{})
	got, err := uc.ListUserPosts(context.Background(), uuid.New(), 0, 0)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestGetFeed_EmptyFollowing(t *testing.T) {
	uc := newPostUC(&mockPostRepo{}, &mockPostCache{}, &mockPublisher{})
	got, err := uc.GetFeed(context.Background(), []uuid.UUID{}, 20, 0)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSearchPosts_ClampsLimit(t *testing.T) {
	repo := &mockPostRepo{}
	repo.On("Search", mock.Anything, "go", 20, 0).Return([]*domain.Post{}, nil)

	uc := newPostUC(repo, &mockPostCache{}, &mockPublisher{})
	_, err := uc.SearchPosts(context.Background(), "go", -5, 0)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCreatePost_PublishError_Ignored(t *testing.T) {
	repo := &mockPostRepo{}
	cache := &mockPostCache{}
	pub := &mockPublisher{}

	repo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pub.On("Publish", mock.Anything, domain.EventPostCreated, mock.Anything).Return(errors.New("nats down"))

	uc := newPostUC(repo, cache, pub)
	p, err := uc.CreatePost(context.Background(), usecase.CreatePostInput{
		AuthorID:  uuid.New(),
		Caption:   "hello",
		MediaURLs: []string{"https://example.com/img.jpg"},
	})
	require.NoError(t, err) // publish errors are non-fatal
	require.NotNil(t, p)
}
