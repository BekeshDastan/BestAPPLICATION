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

type mockHighlightRepo struct{ mock.Mock }

func (m *mockHighlightRepo) Create(ctx context.Context, h *domain.Highlight) error {
	return m.Called(ctx, h).Error(0)
}
func (m *mockHighlightRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Highlight, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Highlight), args.Error(1)
}
func (m *mockHighlightRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockHighlightRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Highlight, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*domain.Highlight), args.Error(1)
}
func (m *mockHighlightRepo) AddStory(ctx context.Context, hs *domain.HighlightStory) error {
	return m.Called(ctx, hs).Error(0)
}
func (m *mockHighlightRepo) RemoveStory(ctx context.Context, highlightID, storyID uuid.UUID) error {
	return m.Called(ctx, highlightID, storyID).Error(0)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func makeHighlight(userID uuid.UUID) *domain.Highlight {
	return &domain.Highlight{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     "My Highlights",
		CreatedAt: time.Now(),
	}
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestCreateHighlight_EmptyTitle(t *testing.T) {
	uc := usecase.NewHighlightUseCase(&mockHighlightRepo{}, &mockStoryRepo{})
	_, err := uc.CreateHighlight(context.Background(), uuid.New(), "  ", "")
	assert.ErrorIs(t, err, domain.ErrInvalidHighlightTitle)
}

func TestCreateHighlight_Success(t *testing.T) {
	highlights := &mockHighlightRepo{}
	highlights.On("Create", mock.Anything, mock.AnythingOfType("*domain.Highlight")).Return(nil)

	uc := usecase.NewHighlightUseCase(highlights, &mockStoryRepo{})
	h, err := uc.CreateHighlight(context.Background(), uuid.New(), "Travel", "")
	require.NoError(t, err)
	assert.Equal(t, "Travel", h.Title)
	highlights.AssertExpectations(t)
}

func TestDeleteHighlight_NotOwner(t *testing.T) {
	highlights := &mockHighlightRepo{}
	ownerID := uuid.New()
	h := makeHighlight(ownerID)
	highlights.On("GetByID", mock.Anything, h.ID).Return(h, nil)

	uc := usecase.NewHighlightUseCase(highlights, &mockStoryRepo{})
	err := uc.DeleteHighlight(context.Background(), h.ID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestDeleteHighlight_Success(t *testing.T) {
	highlights := &mockHighlightRepo{}
	ownerID := uuid.New()
	h := makeHighlight(ownerID)
	highlights.On("GetByID", mock.Anything, h.ID).Return(h, nil)
	highlights.On("Delete", mock.Anything, h.ID).Return(nil)

	uc := usecase.NewHighlightUseCase(highlights, &mockStoryRepo{})
	err := uc.DeleteHighlight(context.Background(), h.ID, ownerID)
	require.NoError(t, err)
	highlights.AssertExpectations(t)
}

func TestAddToHighlight_NotHighlightOwner(t *testing.T) {
	highlights := &mockHighlightRepo{}
	stories := &mockStoryRepo{}
	ownerID := uuid.New()
	h := makeHighlight(ownerID)
	highlights.On("GetByID", mock.Anything, h.ID).Return(h, nil)

	uc := usecase.NewHighlightUseCase(highlights, stories)
	err := uc.AddToHighlight(context.Background(), h.ID, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestAddToHighlight_StoryNotOwned(t *testing.T) {
	highlights := &mockHighlightRepo{}
	stories := &mockStoryRepo{}
	ownerID := uuid.New()
	h := makeHighlight(ownerID)
	s := makeStory(uuid.New())

	highlights.On("GetByID", mock.Anything, h.ID).Return(h, nil)
	stories.On("GetByID", mock.Anything, s.ID).Return(s, nil)

	uc := usecase.NewHighlightUseCase(highlights, stories)
	err := uc.AddToHighlight(context.Background(), h.ID, s.ID, ownerID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestAddToHighlight_Success(t *testing.T) {
	highlights := &mockHighlightRepo{}
	stories := &mockStoryRepo{}
	ownerID := uuid.New()
	h := makeHighlight(ownerID)
	s := makeStory(ownerID)

	highlights.On("GetByID", mock.Anything, h.ID).Return(h, nil)
	stories.On("GetByID", mock.Anything, s.ID).Return(s, nil)
	highlights.On("AddStory", mock.Anything, mock.AnythingOfType("*domain.HighlightStory")).Return(nil)

	uc := usecase.NewHighlightUseCase(highlights, stories)
	err := uc.AddToHighlight(context.Background(), h.ID, s.ID, ownerID)
	require.NoError(t, err)
	highlights.AssertExpectations(t)
}

func TestListHighlights_Success(t *testing.T) {
	highlights := &mockHighlightRepo{}
	userID := uuid.New()
	hs := []*domain.Highlight{makeHighlight(userID)}
	highlights.On("ListByUser", mock.Anything, userID).Return(hs, nil)

	uc := usecase.NewHighlightUseCase(highlights, &mockStoryRepo{})
	got, err := uc.ListHighlights(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, got, 1)
}
