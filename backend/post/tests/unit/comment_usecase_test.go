package unit_test

import (
	"context"
	"strings"
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

type mockCommentRepo struct{ mock.Mock }

func (m *mockCommentRepo) Create(ctx context.Context, c *domain.Comment) error {
	return m.Called(ctx, c).Error(0)
}
func (m *mockCommentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Comment), args.Error(1)
}
func (m *mockCommentRepo) ListByPost(ctx context.Context, postID uuid.UUID, limit, offset int) ([]*domain.Comment, error) {
	args := m.Called(ctx, postID, limit, offset)
	return args.Get(0).([]*domain.Comment), args.Error(1)
}
func (m *mockCommentRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func newCommentUC(postRepo *mockPostRepo, commentRepo *mockCommentRepo, pub *mockPublisher) *usecase.CommentUseCase {
	return usecase.NewCommentUseCase(postRepo, commentRepo, pub, &mockTransactor{})
}

func makeComment(postID, authorID uuid.UUID) *domain.Comment {
	return &domain.Comment{
		ID:        uuid.New(),
		PostID:    postID,
		AuthorID:  authorID,
		Body:      "great photo!",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestAddComment_Success(t *testing.T) {
	postRepo := &mockPostRepo{}
	commentRepo := &mockCommentRepo{}
	pub := &mockPublisher{}

	p := makePost()
	authorID := uuid.New()

	postRepo.On("GetByID", mock.Anything, p.ID).Return(p, nil)
	commentRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Comment")).Return(nil)
	postRepo.On("IncrementComments", mock.Anything, p.ID).Return(nil)
	pub.On("Publish", mock.Anything, domain.EventPostCommented, mock.Anything).Return(nil)

	uc := newCommentUC(postRepo, commentRepo, pub)
	c, err := uc.AddComment(context.Background(), p.ID, authorID, "great photo!")

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, "great photo!", c.Body)
	assert.Equal(t, p.ID, c.PostID)
}

func TestAddComment_EmptyBody(t *testing.T) {
	uc := newCommentUC(&mockPostRepo{}, &mockCommentRepo{}, &mockPublisher{})
	_, err := uc.AddComment(context.Background(), uuid.New(), uuid.New(), "   ")
	assert.ErrorIs(t, err, domain.ErrCommentEmpty)
}

func TestAddComment_TooLong(t *testing.T) {
	postRepo := &mockPostRepo{}
	p := makePost()
	postRepo.On("GetByID", mock.Anything, p.ID).Return(p, nil)

	uc := newCommentUC(postRepo, &mockCommentRepo{}, &mockPublisher{})
	long := strings.Repeat("x", domain.MaxCommentLen+1)
	_, err := uc.AddComment(context.Background(), p.ID, uuid.New(), long)
	assert.ErrorIs(t, err, domain.ErrCommentTooLong)
}

func TestAddComment_PostNotFound(t *testing.T) {
	postRepo := &mockPostRepo{}
	postID := uuid.New()
	postRepo.On("GetByID", mock.Anything, postID).Return(nil, domain.ErrPostNotFound)

	uc := newCommentUC(postRepo, &mockCommentRepo{}, &mockPublisher{})
	_, err := uc.AddComment(context.Background(), postID, uuid.New(), "hello")
	assert.ErrorIs(t, err, domain.ErrPostNotFound)
}

func TestDeleteComment_Success(t *testing.T) {
	postRepo := &mockPostRepo{}
	commentRepo := &mockCommentRepo{}

	p := makePost()
	authorID := uuid.New()
	c := makeComment(p.ID, authorID)

	commentRepo.On("GetByID", mock.Anything, c.ID).Return(c, nil)
	commentRepo.On("SoftDelete", mock.Anything, c.ID).Return(nil)
	postRepo.On("DecrementComments", mock.Anything, p.ID).Return(nil)

	uc := newCommentUC(postRepo, commentRepo, &mockPublisher{})
	err := uc.DeleteComment(context.Background(), c.ID, authorID)
	require.NoError(t, err)
}

func TestDeleteComment_NotOwner(t *testing.T) {
	commentRepo := &mockCommentRepo{}

	c := makeComment(uuid.New(), uuid.New())
	commentRepo.On("GetByID", mock.Anything, c.ID).Return(c, nil)

	uc := newCommentUC(&mockPostRepo{}, commentRepo, &mockPublisher{})
	err := uc.DeleteComment(context.Background(), c.ID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestDeleteComment_NotFound(t *testing.T) {
	commentRepo := &mockCommentRepo{}
	commentID := uuid.New()
	commentRepo.On("GetByID", mock.Anything, commentID).Return(nil, domain.ErrCommentNotFound)

	uc := newCommentUC(&mockPostRepo{}, commentRepo, &mockPublisher{})
	err := uc.DeleteComment(context.Background(), commentID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrCommentNotFound)
}

func TestListComments_Success(t *testing.T) {
	commentRepo := &mockCommentRepo{}
	postID := uuid.New()
	comments := []*domain.Comment{
		makeComment(postID, uuid.New()),
		makeComment(postID, uuid.New()),
	}
	commentRepo.On("ListByPost", mock.Anything, postID, 20, 0).Return(comments, nil)

	uc := newCommentUC(&mockPostRepo{}, commentRepo, &mockPublisher{})
	got, err := uc.ListComments(context.Background(), postID, 0, 0)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestListComments_FiltersDeleted(t *testing.T) {
	commentRepo := &mockCommentRepo{}
	postID := uuid.New()

	active := makeComment(postID, uuid.New())
	deleted := makeComment(postID, uuid.New())
	now := time.Now()
	deleted.DeletedAt = &now

	commentRepo.On("ListByPost", mock.Anything, postID, 20, 0).Return([]*domain.Comment{active, deleted}, nil)

	uc := newCommentUC(&mockPostRepo{}, commentRepo, &mockPublisher{})
	got, err := uc.ListComments(context.Background(), postID, 0, 0)
	require.NoError(t, err)
	assert.Len(t, got, 1) // deleted filtered out
	assert.Equal(t, active.ID, got[0].ID)
}
