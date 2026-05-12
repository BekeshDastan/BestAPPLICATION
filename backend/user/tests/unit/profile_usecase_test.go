package unit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bekesh/social/backend/user/internal/domain"
	"github.com/bekesh/social/backend/user/internal/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ── Helpers ────────────────────────────────────────────────────────────────

func newProfileUC(users *mockUserRepo, cache *mockCache, pub *mockPublisher) *usecase.ProfileUseCase {
	return usecase.NewProfileUseCase(users, cache, pub)
}

func makeUser(opts ...func(*domain.User)) *domain.User {
	u := &domain.User{
		ID:       uuid.New(),
		Email:    "alice@example.com",
		Username: "alice",
		FullName: "Alice Smith",
	}
	for _, o := range opts {
		o(u)
	}
	return u
}

// ── GetProfile ─────────────────────────────────────────────────────────────

func TestGetProfile_CacheHit(t *testing.T) {
	users := &mockUserRepo{}
	cache := &mockCache{}
	cached := makeUser()

	cache.On("GetProfile", mock.Anything, cached.ID).Return(cached, nil)

	uc := newProfileUC(users, cache, &mockPublisher{})
	got, err := uc.GetProfile(context.Background(), cached.ID)

	require.NoError(t, err)
	assert.Equal(t, cached.ID, got.ID)
	users.AssertNotCalled(t, "GetByID")
}

func TestGetProfile_CacheMiss_HitsDB(t *testing.T) {
	users := &mockUserRepo{}
	cache := &mockCache{}
	u := makeUser()

	cache.On("GetProfile", mock.Anything, u.ID).Return(nil, errors.New("cache miss"))
	users.On("GetByID", mock.Anything, u.ID).Return(u, nil)
	cache.On("SetProfile", mock.Anything, u, 5*time.Minute).Return(nil)

	uc := newProfileUC(users, cache, &mockPublisher{})
	got, err := uc.GetProfile(context.Background(), u.ID)

	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)
	users.AssertCalled(t, "GetByID", mock.Anything, u.ID)
}

func TestGetProfile_DeletedUser(t *testing.T) {
	users := &mockUserRepo{}
	cache := &mockCache{}
	deleted := makeUser(func(u *domain.User) {
		now := time.Now()
		u.DeletedAt = &now
	})

	cache.On("GetProfile", mock.Anything, deleted.ID).Return(nil, errors.New("cache miss"))
	users.On("GetByID", mock.Anything, deleted.ID).Return(deleted, nil)

	uc := newProfileUC(users, cache, &mockPublisher{})
	_, err := uc.GetProfile(context.Background(), deleted.ID)

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetProfile_NotFound(t *testing.T) {
	users := &mockUserRepo{}
	cache := &mockCache{}
	id := uuid.New()

	cache.On("GetProfile", mock.Anything, id).Return(nil, errors.New("cache miss"))
	users.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound)

	uc := newProfileUC(users, cache, &mockPublisher{})
	_, err := uc.GetProfile(context.Background(), id)

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// ── GetProfileByUsername ───────────────────────────────────────────────────

func TestGetProfileByUsername_Success(t *testing.T) {
	users := &mockUserRepo{}
	u := makeUser()

	users.On("GetByUsername", mock.Anything, "alice").Return(u, nil)

	uc := newProfileUC(users, &mockCache{}, &mockPublisher{})
	got, err := uc.GetProfileByUsername(context.Background(), "alice")

	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)
}

func TestGetProfileByUsername_DeletedReturnsNotFound(t *testing.T) {
	users := &mockUserRepo{}
	deleted := makeUser(func(u *domain.User) {
		now := time.Now()
		u.DeletedAt = &now
	})

	users.On("GetByUsername", mock.Anything, "alice").Return(deleted, nil)

	uc := newProfileUC(users, &mockCache{}, &mockPublisher{})
	_, err := uc.GetProfileByUsername(context.Background(), "alice")

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// ── UpdateProfile ──────────────────────────────────────────────────────────

func TestUpdateProfile_Success(t *testing.T) {
	users := &mockUserRepo{}
	cache := &mockCache{}
	u := makeUser()

	users.On("GetByID", mock.Anything, u.ID).Return(u, nil)
	users.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)
	cache.On("InvalidateProfile", mock.Anything, u.ID).Return(nil)

	uc := newProfileUC(users, cache, &mockPublisher{})
	in := usecase.UpdateProfileInput{FullName: "Alice Updated", Bio: "new bio", IsPrivate: true}
	got, err := uc.UpdateProfile(context.Background(), u.ID, in)

	require.NoError(t, err)
	assert.Equal(t, "Alice Updated", got.FullName)
	assert.Equal(t, "new bio", got.Bio)
	assert.True(t, got.IsPrivate)
	cache.AssertCalled(t, "InvalidateProfile", mock.Anything, u.ID)
}

func TestUpdateProfile_UserNotFound(t *testing.T) {
	users := &mockUserRepo{}
	id := uuid.New()

	users.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound)

	uc := newProfileUC(users, &mockCache{}, &mockPublisher{})
	_, err := uc.UpdateProfile(context.Background(), id, usecase.UpdateProfileInput{})

	assert.Error(t, err)
}

func TestUpdateProfile_DeletedUserNotFound(t *testing.T) {
	users := &mockUserRepo{}
	cache := &mockCache{}
	deleted := makeUser(func(u *domain.User) {
		now := time.Now()
		u.DeletedAt = &now
	})

	users.On("GetByID", mock.Anything, deleted.ID).Return(deleted, nil)

	uc := newProfileUC(users, cache, &mockPublisher{})
	_, err := uc.UpdateProfile(context.Background(), deleted.ID, usecase.UpdateProfileInput{})

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// ── UpdateAvatar ───────────────────────────────────────────────────────────

func TestUpdateAvatar_Success(t *testing.T) {
	users := &mockUserRepo{}
	cache := &mockCache{}
	u := makeUser()

	users.On("GetByID", mock.Anything, u.ID).Return(u, nil)
	users.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)
	cache.On("InvalidateProfile", mock.Anything, u.ID).Return(nil)

	uc := newProfileUC(users, cache, &mockPublisher{})
	got, err := uc.UpdateAvatar(context.Background(), u.ID, "https://cdn.example.com/avatar.jpg")

	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/avatar.jpg", got.AvatarURL)
}

// ── DeleteAccount ──────────────────────────────────────────────────────────

func TestDeleteAccount_Success(t *testing.T) {
	users := &mockUserRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	hash := testBcryptHash(t, "password123")
	u := &domain.User{ID: uuid.New(), PasswordHash: hash}

	users.On("GetByID", mock.Anything, u.ID).Return(u, nil)
	users.On("SoftDelete", mock.Anything, u.ID).Return(nil)
	cache.On("InvalidateProfile", mock.Anything, u.ID).Return(nil)
	pub.On("Publish", mock.Anything, domain.EventUserDeleted, mock.Anything).Return(nil)

	uc := newProfileUC(users, cache, pub)
	err := uc.DeleteAccount(context.Background(), u.ID, "password123")

	require.NoError(t, err)
	users.AssertCalled(t, "SoftDelete", mock.Anything, u.ID)
}

func TestDeleteAccount_WrongPassword(t *testing.T) {
	users := &mockUserRepo{}

	hash := testBcryptHash(t, "password123")
	u := &domain.User{ID: uuid.New(), PasswordHash: hash}

	users.On("GetByID", mock.Anything, u.ID).Return(u, nil)

	uc := newProfileUC(users, &mockCache{}, &mockPublisher{})
	err := uc.DeleteAccount(context.Background(), u.ID, "wrongpassword")

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

// ── SearchUsers ────────────────────────────────────────────────────────────

func TestSearchUsers_FiltersDeleted(t *testing.T) {
	users := &mockUserRepo{}
	now := time.Now()
	active := makeUser()
	deleted := makeUser(func(u *domain.User) { u.DeletedAt = &now })

	users.On("Search", mock.Anything, "ali", 20, 0).Return([]*domain.User{active, deleted}, nil)

	uc := newProfileUC(users, &mockCache{}, &mockPublisher{})
	results, err := uc.SearchUsers(context.Background(), "ali", 20, 0)

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, active.ID, results[0].ID)
}

func TestSearchUsers_ClampsLimit(t *testing.T) {
	users := &mockUserRepo{}
	// limit=0 gets clamped to 20
	users.On("Search", mock.Anything, "x", 20, 0).Return([]*domain.User{}, nil)

	uc := newProfileUC(users, &mockCache{}, &mockPublisher{})
	_, err := uc.SearchUsers(context.Background(), "x", 0, 0)
	require.NoError(t, err)
	users.AssertCalled(t, "Search", mock.Anything, "x", 20, 0)
}
