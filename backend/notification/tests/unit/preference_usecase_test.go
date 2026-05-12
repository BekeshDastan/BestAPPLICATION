package unit_test

import (
	"context"
	"testing"

	"github.com/bekesh/social/backend/notification/internal/domain"
	"github.com/bekesh/social/backend/notification/internal/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockPrefRepo struct{ mock.Mock }

func (m *mockPrefRepo) GetAll(ctx context.Context, userID uuid.UUID) ([]*domain.NotificationPreference, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*domain.NotificationPreference), args.Error(1)
}
func (m *mockPrefRepo) Upsert(ctx context.Context, p *domain.NotificationPreference) error {
	return m.Called(ctx, p).Error(0)
}

func TestGetPreferences_Success(t *testing.T) {
	repo := &mockPrefRepo{}
	userID := uuid.New()

	prefs := []*domain.NotificationPreference{
		{UserID: userID, Type: domain.NotificationTypeLike, EmailEnabled: true, PushEnabled: true},
	}
	repo.On("GetAll", mock.Anything, userID).Return(prefs, nil)

	uc := usecase.NewPreferenceUseCase(repo)
	got, err := uc.GetPreferences(context.Background(), userID)

	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, domain.NotificationTypeLike, got[0].Type)
}

func TestUpdatePreference_Success(t *testing.T) {
	repo := &mockPrefRepo{}
	userID := uuid.New()

	p := &domain.NotificationPreference{
		UserID:       userID,
		Type:         domain.NotificationTypeFollow,
		EmailEnabled: false,
		PushEnabled:  true,
	}
	repo.On("Upsert", mock.Anything, p).Return(nil)

	uc := usecase.NewPreferenceUseCase(repo)
	err := uc.UpdatePreference(context.Background(), p)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdatePreference_EmptyUserID(t *testing.T) {
	repo := &mockPrefRepo{}

	p := &domain.NotificationPreference{
		UserID: uuid.Nil,
		Type:   domain.NotificationTypeLike,
	}

	uc := usecase.NewPreferenceUseCase(repo)
	err := uc.UpdatePreference(context.Background(), p)

	assert.ErrorIs(t, err, domain.ErrUserIDRequired)
}
