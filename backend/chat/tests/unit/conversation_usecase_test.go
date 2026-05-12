package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/bekesh/social/backend/chat/internal/domain"
	"github.com/bekesh/social/backend/chat/internal/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ── Mocks ──────────────────────────────────────────────────────────────────

type mockConvRepo struct{ mock.Mock }

func (m *mockConvRepo) Create(ctx context.Context, conv *domain.Conversation) error {
	return m.Called(ctx, conv).Error(0)
}
func (m *mockConvRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Conversation), args.Error(1)
}
func (m *mockConvRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockConvRepo) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Conversation, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]*domain.Conversation), args.Error(1)
}
func (m *mockConvRepo) UpdateLastMessageAt(ctx context.Context, id uuid.UUID, t time.Time) error {
	return m.Called(ctx, id, t).Error(0)
}
func (m *mockConvRepo) UpdateInfo(ctx context.Context, id uuid.UUID, name, avatarURL string) error {
	return m.Called(ctx, id, name, avatarURL).Error(0)
}

type mockPartRepo struct{ mock.Mock }

func (m *mockPartRepo) Add(ctx context.Context, p *domain.Participant) error {
	return m.Called(ctx, p).Error(0)
}
func (m *mockPartRepo) Remove(ctx context.Context, convID, userID uuid.UUID) error {
	return m.Called(ctx, convID, userID).Error(0)
}
func (m *mockPartRepo) IsParticipant(ctx context.Context, convID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, convID, userID)
	return args.Bool(0), args.Error(1)
}
func (m *mockPartRepo) ListParticipants(ctx context.Context, convID uuid.UUID) ([]*domain.Participant, error) {
	args := m.Called(ctx, convID)
	return args.Get(0).([]*domain.Participant), args.Error(1)
}
func (m *mockPartRepo) GetParticipant(ctx context.Context, convID, userID uuid.UUID) (*domain.Participant, error) {
	args := m.Called(ctx, convID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Participant), args.Error(1)
}
func (m *mockPartRepo) MarkRead(ctx context.Context, convID, userID uuid.UUID) error {
	return m.Called(ctx, convID, userID).Error(0)
}
func (m *mockPartRepo) IncrUnreadExceptSender(ctx context.Context, convID, senderID uuid.UUID) error {
	return m.Called(ctx, convID, senderID).Error(0)
}

type mockPublisher struct{ mock.Mock }

func (m *mockPublisher) Publish(ctx context.Context, subject string, payload any) error {
	return m.Called(ctx, subject, payload).Error(0)
}

type mockCache struct{ mock.Mock }

func (m *mockCache) SetTyping(ctx context.Context, convID, userID uuid.UUID, ttl time.Duration) error {
	return m.Called(ctx, convID, userID, ttl).Error(0)
}
func (m *mockCache) IsTyping(ctx context.Context, convID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, convID, userID)
	return args.Bool(0), args.Error(1)
}
func (m *mockCache) SetOnline(ctx context.Context, userID uuid.UUID, ttl time.Duration) error {
	return m.Called(ctx, userID, ttl).Error(0)
}
func (m *mockCache) IsOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}
func (m *mockCache) IncrUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockCache) GetUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockCache) DelUnread(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

type mockTransactor struct{}

func (m *mockTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func newConvUC(convRepo *mockConvRepo, partRepo *mockPartRepo, pub *mockPublisher, cache *mockCache) *usecase.ConversationUseCase {
	return usecase.NewConversationUseCase(convRepo, partRepo, pub, cache, &mockTransactor{})
}

func makeConv(convType domain.ConversationType, creatorID uuid.UUID) *domain.Conversation {
	return &domain.Conversation{
		ID:        uuid.New(),
		Type:      convType,
		CreatedBy: creatorID,
		CreatedAt: time.Now(),
	}
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestCreateConversation_Success(t *testing.T) {
	convRepo := &mockConvRepo{}
	partRepo := &mockPartRepo{}
	pub := &mockPublisher{}
	cache := &mockCache{}

	creatorID, memberID := uuid.New(), uuid.New()

	convRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Conversation")).Return(nil)
	partRepo.On("Add", mock.Anything, mock.AnythingOfType("*domain.Participant")).Return(nil)

	uc := newConvUC(convRepo, partRepo, pub, cache)
	conv, err := uc.CreateConversation(context.Background(), creatorID, []uuid.UUID{memberID})

	require.NoError(t, err)
	require.NotNil(t, conv)
	assert.Equal(t, domain.ConvTypeDirect, conv.Type)
	assert.Equal(t, creatorID, conv.CreatedBy)
	convRepo.AssertExpectations(t)
	partRepo.AssertExpectations(t)
}

func TestCreateGroupChat_Success(t *testing.T) {
	convRepo := &mockConvRepo{}
	partRepo := &mockPartRepo{}

	creatorID := uuid.New()
	members := []uuid.UUID{uuid.New(), uuid.New()}

	convRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Conversation")).Return(nil)
	partRepo.On("Add", mock.Anything, mock.AnythingOfType("*domain.Participant")).Return(nil)

	uc := newConvUC(convRepo, partRepo, &mockPublisher{}, &mockCache{})
	conv, err := uc.CreateGroupChat(context.Background(), creatorID, "Team Chat", members)

	require.NoError(t, err)
	require.NotNil(t, conv)
	assert.Equal(t, domain.ConvTypeGroup, conv.Type)
	assert.Equal(t, "Team Chat", conv.Name)
}

func TestCreateGroupChat_EmptyName(t *testing.T) {
	uc := newConvUC(&mockConvRepo{}, &mockPartRepo{}, &mockPublisher{}, &mockCache{})
	_, err := uc.CreateGroupChat(context.Background(), uuid.New(), "  ", []uuid.UUID{uuid.New()})
	assert.ErrorIs(t, err, domain.ErrInvalidGroupName)
}

func TestGetConversation_NotParticipant(t *testing.T) {
	convRepo := &mockConvRepo{}
	partRepo := &mockPartRepo{}

	convID, userID := uuid.New(), uuid.New()
	partRepo.On("IsParticipant", mock.Anything, convID, userID).Return(false, nil)

	uc := newConvUC(convRepo, partRepo, &mockPublisher{}, &mockCache{})
	_, err := uc.GetConversation(context.Background(), convID, userID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestGetConversation_Success(t *testing.T) {
	convRepo := &mockConvRepo{}
	partRepo := &mockPartRepo{}

	creatorID := uuid.New()
	conv := makeConv(domain.ConvTypeDirect, creatorID)

	partRepo.On("IsParticipant", mock.Anything, conv.ID, creatorID).Return(true, nil)
	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)

	uc := newConvUC(convRepo, partRepo, &mockPublisher{}, &mockCache{})
	got, err := uc.GetConversation(context.Background(), conv.ID, creatorID)
	require.NoError(t, err)
	assert.Equal(t, conv.ID, got.ID)
}

func TestDeleteConversation_NotCreator(t *testing.T) {
	convRepo := &mockConvRepo{}

	creatorID := uuid.New()
	conv := makeConv(domain.ConvTypeGroup, creatorID)
	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)

	uc := newConvUC(convRepo, &mockPartRepo{}, &mockPublisher{}, &mockCache{})
	err := uc.DeleteConversation(context.Background(), conv.ID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestDeleteConversation_Success(t *testing.T) {
	convRepo := &mockConvRepo{}

	creatorID := uuid.New()
	conv := makeConv(domain.ConvTypeDirect, creatorID)
	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)
	convRepo.On("Delete", mock.Anything, conv.ID).Return(nil)

	uc := newConvUC(convRepo, &mockPartRepo{}, &mockPublisher{}, &mockCache{})
	err := uc.DeleteConversation(context.Background(), conv.ID, creatorID)
	require.NoError(t, err)
	convRepo.AssertExpectations(t)
}

func TestMarkConversationRead_NotParticipant(t *testing.T) {
	partRepo := &mockPartRepo{}
	convID, userID := uuid.New(), uuid.New()
	partRepo.On("IsParticipant", mock.Anything, convID, userID).Return(false, nil)

	uc := newConvUC(&mockConvRepo{}, partRepo, &mockPublisher{}, &mockCache{})
	err := uc.MarkConversationRead(context.Background(), convID, userID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestMarkConversationRead_Success(t *testing.T) {
	partRepo := &mockPartRepo{}
	cache := &mockCache{}

	convID, userID := uuid.New(), uuid.New()
	partRepo.On("IsParticipant", mock.Anything, convID, userID).Return(true, nil)
	partRepo.On("MarkRead", mock.Anything, convID, userID).Return(nil)
	cache.On("DelUnread", mock.Anything, userID).Return(nil)

	uc := newConvUC(&mockConvRepo{}, partRepo, &mockPublisher{}, cache)
	err := uc.MarkConversationRead(context.Background(), convID, userID)
	require.NoError(t, err)
}

func TestAddParticipant_NotGroup(t *testing.T) {
	convRepo := &mockConvRepo{}
	creatorID := uuid.New()
	conv := makeConv(domain.ConvTypeDirect, creatorID)
	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)

	uc := newConvUC(convRepo, &mockPartRepo{}, &mockPublisher{}, &mockCache{})
	err := uc.AddParticipant(context.Background(), conv.ID, creatorID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrGroupRequired)
}

func TestLeaveGroup_Success(t *testing.T) {
	convRepo := &mockConvRepo{}
	partRepo := &mockPartRepo{}

	creatorID, memberID := uuid.New(), uuid.New()
	conv := makeConv(domain.ConvTypeGroup, creatorID)
	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)
	partRepo.On("IsParticipant", mock.Anything, conv.ID, memberID).Return(true, nil)
	partRepo.On("Remove", mock.Anything, conv.ID, memberID).Return(nil)

	uc := newConvUC(convRepo, partRepo, &mockPublisher{}, &mockCache{})
	err := uc.LeaveGroup(context.Background(), conv.ID, memberID)
	require.NoError(t, err)
}

func TestListConversations_Success(t *testing.T) {
	convRepo := &mockConvRepo{}
	userID := uuid.New()
	convs := []*domain.Conversation{makeConv(domain.ConvTypeDirect, userID)}
	convRepo.On("ListByUser", mock.Anything, userID, 20, 0).Return(convs, nil)

	uc := newConvUC(convRepo, &mockPartRepo{}, &mockPublisher{}, &mockCache{})
	got, err := uc.ListConversations(context.Background(), userID, 0, 0)
	require.NoError(t, err)
	assert.Len(t, got, 1)
}
