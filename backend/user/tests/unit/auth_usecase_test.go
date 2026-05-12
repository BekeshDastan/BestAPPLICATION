package unit_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/bekesh/social/backend/user/internal/domain"
	"github.com/bekesh/social/backend/user/internal/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// ── Mocks ──────────────────────────────────────────────────────────────────

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) Create(ctx context.Context, u *domain.User) error {
	return m.Called(ctx, u).Error(0)
}
func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) Update(ctx context.Context, u *domain.User) error {
	return m.Called(ctx, u).Error(0)
}
func (m *mockUserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockUserRepo) Search(ctx context.Context, q string, limit, offset int) ([]*domain.User, error) {
	args := m.Called(ctx, q, limit, offset)
	return args.Get(0).([]*domain.User), args.Error(1)
}
func (m *mockUserRepo) CountFollowers(ctx context.Context, id uuid.UUID) (int, error) {
	args := m.Called(ctx, id)
	return args.Int(0), args.Error(1)
}
func (m *mockUserRepo) CountFollowing(ctx context.Context, id uuid.UUID) (int, error) {
	args := m.Called(ctx, id)
	return args.Int(0), args.Error(1)
}

type mockTokenRepo struct{ mock.Mock }

func (m *mockTokenRepo) SaveRefreshToken(ctx context.Context, t *domain.RefreshToken) error {
	return m.Called(ctx, t).Error(0)
}
func (m *mockTokenRepo) GetRefreshToken(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.RefreshToken), args.Error(1)
}
func (m *mockTokenRepo) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockTokenRepo) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}
func (m *mockTokenRepo) SaveEmailVerification(ctx context.Context, ev *domain.EmailVerification) error {
	return m.Called(ctx, ev).Error(0)
}
func (m *mockTokenRepo) GetEmailVerification(ctx context.Context, token string) (*domain.EmailVerification, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.EmailVerification), args.Error(1)
}
func (m *mockTokenRepo) DeleteEmailVerification(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}
func (m *mockTokenRepo) SavePasswordReset(ctx context.Context, pr *domain.PasswordReset) error {
	return m.Called(ctx, pr).Error(0)
}
func (m *mockTokenRepo) GetPasswordReset(ctx context.Context, token string) (*domain.PasswordReset, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PasswordReset), args.Error(1)
}
func (m *mockTokenRepo) MarkPasswordResetUsed(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}

type mockCache struct{ mock.Mock }

func (m *mockCache) GetProfile(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockCache) SetProfile(ctx context.Context, u *domain.User, ttl time.Duration) error {
	return m.Called(ctx, u, ttl).Error(0)
}
func (m *mockCache) InvalidateProfile(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockCache) IsTokenBlacklisted(ctx context.Context, hash string) (bool, error) {
	args := m.Called(ctx, hash)
	return args.Bool(0), args.Error(1)
}
func (m *mockCache) BlacklistToken(ctx context.Context, hash string, ttl time.Duration) error {
	return m.Called(ctx, hash, ttl).Error(0)
}

type mockPublisher struct{ mock.Mock }

func (m *mockPublisher) Publish(ctx context.Context, subject string, payload any) error {
	return m.Called(ctx, subject, payload).Error(0)
}

// mockTransactor runs fn inline — no real DB tx in unit tests.
type mockTransactor struct{}

func (m *mockTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func testKeys(t *testing.T) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return priv, &priv.PublicKey
}

func newAuthUC(t *testing.T, users *mockUserRepo, tokens *mockTokenRepo, cache *mockCache, pub *mockPublisher) *usecase.AuthUseCase {
	t.Helper()
	priv, pubKey := testKeys(t)
	return usecase.NewAuthUseCase(users, tokens, cache, pub, &mockTransactor{}, priv, pubKey, 15*time.Minute, 7*24*time.Hour)
}

func testBcryptHash(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	users := &mockUserRepo{}
	tokens := &mockTokenRepo{}
	cache := &mockCache{}
	pub := &mockPublisher{}

	users.On("GetByEmail", mock.Anything, "alice@example.com").Return(nil, domain.ErrNotFound)
	users.On("GetByUsername", mock.Anything, "alice").Return(nil, domain.ErrNotFound)
	users.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)
	tokens.On("SaveEmailVerification", mock.Anything, mock.AnythingOfType("*domain.EmailVerification")).Return(nil)
	tokens.On("SaveRefreshToken", mock.Anything, mock.AnythingOfType("*domain.RefreshToken")).Return(nil)
	pub.On("Publish", mock.Anything, domain.EventUserRegistered, mock.Anything).Return(nil)

	uc := newAuthUC(t, users, tokens, cache, pub)
	user, pair, err := uc.Register(context.Background(), "alice@example.com", "alice", "password123", "Alice")

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, domain.Username("alice"), user.Username)
	assert.False(t, user.IsVerified)
	pub.AssertCalled(t, "Publish", mock.Anything, domain.EventUserRegistered, mock.Anything)
}

func TestRegister_EmailTaken(t *testing.T) {
	users := &mockUserRepo{}
	existing := &domain.User{ID: uuid.New(), Email: "alice@example.com"}
	users.On("GetByEmail", mock.Anything, "alice@example.com").Return(existing, nil)

	uc := newAuthUC(t, users, &mockTokenRepo{}, &mockCache{}, &mockPublisher{})
	_, _, err := uc.Register(context.Background(), "alice@example.com", "alice", "password123", "Alice")
	assert.ErrorIs(t, err, domain.ErrEmailTaken)
}

func TestRegister_UsernameTaken(t *testing.T) {
	users := &mockUserRepo{}
	users.On("GetByEmail", mock.Anything, "alice@example.com").Return(nil, domain.ErrNotFound)
	existing := &domain.User{ID: uuid.New(), Username: "alice"}
	users.On("GetByUsername", mock.Anything, "alice").Return(existing, nil)

	uc := newAuthUC(t, users, &mockTokenRepo{}, &mockCache{}, &mockPublisher{})
	_, _, err := uc.Register(context.Background(), "alice@example.com", "alice", "password123", "Alice")
	assert.ErrorIs(t, err, domain.ErrUsernameTaken)
}

func TestRegister_WeakPassword(t *testing.T) {
	uc := newAuthUC(t, &mockUserRepo{}, &mockTokenRepo{}, &mockCache{}, &mockPublisher{})
	_, _, err := uc.Register(context.Background(), "alice@example.com", "alice", "short", "Alice")
	assert.ErrorIs(t, err, domain.ErrWeakPassword)
}

func TestRegister_InvalidEmail(t *testing.T) {
	uc := newAuthUC(t, &mockUserRepo{}, &mockTokenRepo{}, &mockCache{}, &mockPublisher{})
	_, _, err := uc.Register(context.Background(), "not-an-email", "alice", "password123", "Alice")
	assert.ErrorIs(t, err, domain.ErrInvalidEmail)
}

func TestLogin_Success(t *testing.T) {
	users := &mockUserRepo{}
	tokens := &mockTokenRepo{}

	storedUser := &domain.User{
		ID:           uuid.New(),
		Email:        "alice@example.com",
		Username:     "alice",
		PasswordHash: testBcryptHash(t, "password123"),
		IsVerified:   true,
	}
	users.On("GetByEmail", mock.Anything, "alice@example.com").Return(storedUser, nil)
	tokens.On("SaveRefreshToken", mock.Anything, mock.AnythingOfType("*domain.RefreshToken")).Return(nil)

	uc := newAuthUC(t, users, tokens, &mockCache{}, &mockPublisher{})
	user, pair, err := uc.Login(context.Background(), "alice@example.com", "password123")

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
}

func TestLogin_WrongPassword(t *testing.T) {
	users := &mockUserRepo{}
	storedUser := &domain.User{
		ID:           uuid.New(),
		PasswordHash: testBcryptHash(t, "password123"),
		IsVerified:   true,
	}
	users.On("GetByEmail", mock.Anything, "alice@example.com").Return(storedUser, nil)

	uc := newAuthUC(t, users, &mockTokenRepo{}, &mockCache{}, &mockPublisher{})
	_, _, err := uc.Login(context.Background(), "alice@example.com", "wrongpassword")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLogin_EmailNotVerified(t *testing.T) {
	users := &mockUserRepo{}
	storedUser := &domain.User{
		ID:           uuid.New(),
		PasswordHash: testBcryptHash(t, "password123"),
		IsVerified:   false,
	}
	users.On("GetByEmail", mock.Anything, "alice@example.com").Return(storedUser, nil)

	uc := newAuthUC(t, users, &mockTokenRepo{}, &mockCache{}, &mockPublisher{})
	_, _, err := uc.Login(context.Background(), "alice@example.com", "password123")
	assert.ErrorIs(t, err, domain.ErrEmailNotVerified)
}

func TestLogin_UserNotFound_ReturnsInvalidCredentials(t *testing.T) {
	users := &mockUserRepo{}
	users.On("GetByEmail", mock.Anything, "nobody@example.com").Return(nil, domain.ErrNotFound)

	uc := newAuthUC(t, users, &mockTokenRepo{}, &mockCache{}, &mockPublisher{})
	_, _, err := uc.Login(context.Background(), "nobody@example.com", "password123")
	// Must NOT leak that user doesn't exist
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestVerifyEmail_Success(t *testing.T) {
	users := &mockUserRepo{}
	tokens := &mockTokenRepo{}
	cache := &mockCache{}

	userID := uuid.New()
	ev := &domain.EmailVerification{
		Token:     "validtoken",
		UserID:    userID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	storedUser := &domain.User{ID: userID, IsVerified: false}

	tokens.On("GetEmailVerification", mock.Anything, "validtoken").Return(ev, nil)
	users.On("GetByID", mock.Anything, userID).Return(storedUser, nil)
	users.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)
	tokens.On("DeleteEmailVerification", mock.Anything, "validtoken").Return(nil)
	cache.On("InvalidateProfile", mock.Anything, userID).Return(nil)

	uc := newAuthUC(t, users, tokens, cache, &mockPublisher{})
	err := uc.VerifyEmail(context.Background(), "validtoken")
	require.NoError(t, err)
	users.AssertCalled(t, "Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
		return u.IsVerified
	}))
}

func TestVerifyEmail_ExpiredToken(t *testing.T) {
	tokens := &mockTokenRepo{}
	ev := &domain.EmailVerification{
		Token:     "expiredtoken",
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	tokens.On("GetEmailVerification", mock.Anything, "expiredtoken").Return(ev, nil)

	uc := newAuthUC(t, &mockUserRepo{}, tokens, &mockCache{}, &mockPublisher{})
	err := uc.VerifyEmail(context.Background(), "expiredtoken")
	assert.ErrorIs(t, err, domain.ErrTokenExpired)
}

func TestRefreshToken_Rotation(t *testing.T) {
	users := &mockUserRepo{}
	tokens := &mockTokenRepo{}
	cache := &mockCache{}

	userID := uuid.New()
	rtID := uuid.New()

	cache.On("IsTokenBlacklisted", mock.Anything, mock.Anything).Return(false, nil)
	tokens.On("GetRefreshToken", mock.Anything, mock.Anything).Return(&domain.RefreshToken{
		ID:        rtID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}, nil)
	users.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID}, nil)
	tokens.On("RevokeRefreshToken", mock.Anything, rtID).Return(nil)
	cache.On("BlacklistToken", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	tokens.On("SaveRefreshToken", mock.Anything, mock.AnythingOfType("*domain.RefreshToken")).Return(nil)

	uc := newAuthUC(t, users, tokens, cache, &mockPublisher{})
	pair, err := uc.RefreshToken(context.Background(), "somerawtoken")
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	tokens.AssertCalled(t, "RevokeRefreshToken", mock.Anything, rtID)
}

func TestLogout_RevokesToken(t *testing.T) {
	tokens := &mockTokenRepo{}
	cache := &mockCache{}
	rtID := uuid.New()

	tokens.On("GetRefreshToken", mock.Anything, mock.Anything).Return(&domain.RefreshToken{
		ID:        rtID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}, nil)
	tokens.On("RevokeRefreshToken", mock.Anything, rtID).Return(nil)
	cache.On("BlacklistToken", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	uc := newAuthUC(t, &mockUserRepo{}, tokens, cache, &mockPublisher{})
	err := uc.Logout(context.Background(), "somerawtoken")
	require.NoError(t, err)
	tokens.AssertCalled(t, "RevokeRefreshToken", mock.Anything, rtID)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	uc := newAuthUC(t, &mockUserRepo{}, &mockTokenRepo{}, &mockCache{}, &mockPublisher{})
	_, err := uc.ValidateToken(context.Background(), "not.a.valid.jwt")
	assert.ErrorIs(t, err, domain.ErrTokenInvalid)
}
