package unit_test

import (
	"context"
	"testing"

	"github.com/bekesh/social/backend/user/internal/domain"
	"github.com/bekesh/social/backend/user/internal/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ── mockFollowRepo ─────────────────────────────────────────────────────────

type mockFollowRepo struct{ mock.Mock }

func (m *mockFollowRepo) Follow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	return m.Called(ctx, followerID, followeeID).Error(0)
}
func (m *mockFollowRepo) Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	return m.Called(ctx, followerID, followeeID).Error(0)
}
func (m *mockFollowRepo) IsFollowing(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error) {
	args := m.Called(ctx, followerID, followeeID)
	return args.Bool(0), args.Error(1)
}
func (m *mockFollowRepo) ListFollowers(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.User, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]*domain.User), args.Error(1)
}
func (m *mockFollowRepo) ListFollowing(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.User, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]*domain.User), args.Error(1)
}
func (m *mockFollowRepo) Block(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	return m.Called(ctx, blockerID, blockedID).Error(0)
}
func (m *mockFollowRepo) Unblock(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	return m.Called(ctx, blockerID, blockedID).Error(0)
}
func (m *mockFollowRepo) IsBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error) {
	args := m.Called(ctx, blockerID, blockedID)
	return args.Bool(0), args.Error(1)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func newFollowUC(users *mockUserRepo, follows *mockFollowRepo, pub *mockPublisher) *usecase.FollowUseCase {
	return usecase.NewFollowUseCase(users, follows, pub)
}

// ── Follow ─────────────────────────────────────────────────────────────────

func TestFollow_Success(t *testing.T) {
	users := &mockUserRepo{}
	follows := &mockFollowRepo{}
	pub := &mockPublisher{}

	followerID := uuid.New()
	followeeID := uuid.New()
	target := &domain.User{ID: followeeID}

	users.On("GetByID", mock.Anything, followeeID).Return(target, nil)
	follows.On("IsBlocked", mock.Anything, followeeID, followerID).Return(false, nil)
	follows.On("IsFollowing", mock.Anything, followerID, followeeID).Return(false, nil)
	follows.On("Follow", mock.Anything, followerID, followeeID).Return(nil)
	pub.On("Publish", mock.Anything, domain.EventUserFollowed, mock.Anything).Return(nil)

	uc := newFollowUC(users, follows, pub)
	err := uc.Follow(context.Background(), followerID, followeeID)

	require.NoError(t, err)
	follows.AssertCalled(t, "Follow", mock.Anything, followerID, followeeID)
	pub.AssertCalled(t, "Publish", mock.Anything, domain.EventUserFollowed, mock.Anything)
}

func TestFollow_SelfFollow(t *testing.T) {
	id := uuid.New()
	uc := newFollowUC(&mockUserRepo{}, &mockFollowRepo{}, &mockPublisher{})
	err := uc.Follow(context.Background(), id, id)
	assert.ErrorIs(t, err, domain.ErrSelfFollow)
}

func TestFollow_AlreadyFollowing(t *testing.T) {
	users := &mockUserRepo{}
	follows := &mockFollowRepo{}

	followerID := uuid.New()
	followeeID := uuid.New()
	target := &domain.User{ID: followeeID}

	users.On("GetByID", mock.Anything, followeeID).Return(target, nil)
	follows.On("IsBlocked", mock.Anything, followeeID, followerID).Return(false, nil)
	follows.On("IsFollowing", mock.Anything, followerID, followeeID).Return(true, nil)

	uc := newFollowUC(users, follows, &mockPublisher{})
	err := uc.Follow(context.Background(), followerID, followeeID)
	assert.ErrorIs(t, err, domain.ErrAlreadyFollowing)
}

func TestFollow_BlockedByTarget(t *testing.T) {
	users := &mockUserRepo{}
	follows := &mockFollowRepo{}

	followerID := uuid.New()
	followeeID := uuid.New()
	target := &domain.User{ID: followeeID}

	users.On("GetByID", mock.Anything, followeeID).Return(target, nil)
	// followee has blocked the follower
	follows.On("IsBlocked", mock.Anything, followeeID, followerID).Return(true, nil)

	uc := newFollowUC(users, follows, &mockPublisher{})
	err := uc.Follow(context.Background(), followerID, followeeID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestFollow_TargetNotFound(t *testing.T) {
	users := &mockUserRepo{}
	followerID := uuid.New()
	followeeID := uuid.New()

	users.On("GetByID", mock.Anything, followeeID).Return(nil, domain.ErrNotFound)

	uc := newFollowUC(users, &mockFollowRepo{}, &mockPublisher{})
	err := uc.Follow(context.Background(), followerID, followeeID)
	assert.Error(t, err)
}

// ── Unfollow ───────────────────────────────────────────────────────────────

func TestUnfollow_Success(t *testing.T) {
	follows := &mockFollowRepo{}
	pub := &mockPublisher{}

	followerID := uuid.New()
	followeeID := uuid.New()

	follows.On("IsFollowing", mock.Anything, followerID, followeeID).Return(true, nil)
	follows.On("Unfollow", mock.Anything, followerID, followeeID).Return(nil)
	pub.On("Publish", mock.Anything, domain.EventUserUnfollowed, mock.Anything).Return(nil)

	uc := newFollowUC(&mockUserRepo{}, follows, pub)
	err := uc.Unfollow(context.Background(), followerID, followeeID)

	require.NoError(t, err)
	follows.AssertCalled(t, "Unfollow", mock.Anything, followerID, followeeID)
}

func TestUnfollow_SelfFollow(t *testing.T) {
	id := uuid.New()
	uc := newFollowUC(&mockUserRepo{}, &mockFollowRepo{}, &mockPublisher{})
	err := uc.Unfollow(context.Background(), id, id)
	assert.ErrorIs(t, err, domain.ErrSelfFollow)
}

func TestUnfollow_NotFollowing(t *testing.T) {
	follows := &mockFollowRepo{}

	followerID := uuid.New()
	followeeID := uuid.New()

	follows.On("IsFollowing", mock.Anything, followerID, followeeID).Return(false, nil)

	uc := newFollowUC(&mockUserRepo{}, follows, &mockPublisher{})
	err := uc.Unfollow(context.Background(), followerID, followeeID)
	assert.ErrorIs(t, err, domain.ErrNotFollowing)
}

// ── IsFollowing ────────────────────────────────────────────────────────────

func TestIsFollowing_ReturnsTrue(t *testing.T) {
	follows := &mockFollowRepo{}
	a, b := uuid.New(), uuid.New()

	follows.On("IsFollowing", mock.Anything, a, b).Return(true, nil)

	uc := newFollowUC(&mockUserRepo{}, follows, &mockPublisher{})
	ok, err := uc.IsFollowing(context.Background(), a, b)

	require.NoError(t, err)
	assert.True(t, ok)
}

// ── ListFollowers / ListFollowing ──────────────────────────────────────────

func TestListFollowers_Success(t *testing.T) {
	follows := &mockFollowRepo{}
	userID := uuid.New()
	list := []*domain.User{{ID: uuid.New()}, {ID: uuid.New()}}

	follows.On("ListFollowers", mock.Anything, userID, 20, 0).Return(list, nil)

	uc := newFollowUC(&mockUserRepo{}, follows, &mockPublisher{})
	got, err := uc.ListFollowers(context.Background(), userID, 20, 0)

	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestListFollowers_ClampsLimit(t *testing.T) {
	follows := &mockFollowRepo{}
	userID := uuid.New()

	// limit=0 should be clamped to 20
	follows.On("ListFollowers", mock.Anything, userID, 20, 0).Return([]*domain.User{}, nil)

	uc := newFollowUC(&mockUserRepo{}, follows, &mockPublisher{})
	_, err := uc.ListFollowers(context.Background(), userID, 0, 0)
	require.NoError(t, err)
	follows.AssertCalled(t, "ListFollowers", mock.Anything, userID, 20, 0)
}

func TestListFollowing_Success(t *testing.T) {
	follows := &mockFollowRepo{}
	userID := uuid.New()
	list := []*domain.User{{ID: uuid.New()}}

	follows.On("ListFollowing", mock.Anything, userID, 10, 0).Return(list, nil)

	uc := newFollowUC(&mockUserRepo{}, follows, &mockPublisher{})
	got, err := uc.ListFollowing(context.Background(), userID, 10, 0)

	require.NoError(t, err)
	assert.Len(t, got, 1)
}

// ── BlockUser ──────────────────────────────────────────────────────────────

func TestBlockUser_Success(t *testing.T) {
	users := &mockUserRepo{}
	follows := &mockFollowRepo{}

	blockerID := uuid.New()
	blockedID := uuid.New()
	target := &domain.User{ID: blockedID}

	users.On("GetByID", mock.Anything, blockedID).Return(target, nil)
	follows.On("IsBlocked", mock.Anything, blockerID, blockedID).Return(false, nil)
	// unfollow both directions silently (returns ignored)
	follows.On("Unfollow", mock.Anything, blockerID, blockedID).Return(nil)
	follows.On("Unfollow", mock.Anything, blockedID, blockerID).Return(nil)
	follows.On("Block", mock.Anything, blockerID, blockedID).Return(nil)

	uc := newFollowUC(users, follows, &mockPublisher{})
	err := uc.BlockUser(context.Background(), blockerID, blockedID)

	require.NoError(t, err)
	follows.AssertCalled(t, "Block", mock.Anything, blockerID, blockedID)
}

func TestBlockUser_SelfBlock(t *testing.T) {
	id := uuid.New()
	uc := newFollowUC(&mockUserRepo{}, &mockFollowRepo{}, &mockPublisher{})
	err := uc.BlockUser(context.Background(), id, id)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestBlockUser_AlreadyBlocked(t *testing.T) {
	users := &mockUserRepo{}
	follows := &mockFollowRepo{}

	blockerID := uuid.New()
	blockedID := uuid.New()
	target := &domain.User{ID: blockedID}

	users.On("GetByID", mock.Anything, blockedID).Return(target, nil)
	follows.On("IsBlocked", mock.Anything, blockerID, blockedID).Return(true, nil)

	uc := newFollowUC(users, follows, &mockPublisher{})
	err := uc.BlockUser(context.Background(), blockerID, blockedID)
	assert.ErrorIs(t, err, domain.ErrAlreadyBlocked)
}

// ── UnblockUser ────────────────────────────────────────────────────────────

func TestUnblockUser_Success(t *testing.T) {
	follows := &mockFollowRepo{}

	blockerID := uuid.New()
	blockedID := uuid.New()

	follows.On("IsBlocked", mock.Anything, blockerID, blockedID).Return(true, nil)
	follows.On("Unblock", mock.Anything, blockerID, blockedID).Return(nil)

	uc := newFollowUC(&mockUserRepo{}, follows, &mockPublisher{})
	err := uc.UnblockUser(context.Background(), blockerID, blockedID)

	require.NoError(t, err)
	follows.AssertCalled(t, "Unblock", mock.Anything, blockerID, blockedID)
}

func TestUnblockUser_NotBlocked(t *testing.T) {
	follows := &mockFollowRepo{}

	blockerID := uuid.New()
	blockedID := uuid.New()

	follows.On("IsBlocked", mock.Anything, blockerID, blockedID).Return(false, nil)

	uc := newFollowUC(&mockUserRepo{}, follows, &mockPublisher{})
	err := uc.UnblockUser(context.Background(), blockerID, blockedID)
	assert.ErrorIs(t, err, domain.ErrNotBlocked)
}
