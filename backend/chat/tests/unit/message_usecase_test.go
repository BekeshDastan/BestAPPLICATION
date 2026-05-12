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

type mockMsgRepo struct{ mock.Mock }

func (m *mockMsgRepo) Create(ctx context.Context, msg *domain.Message) error {
	return m.Called(ctx, msg).Error(0)
}
func (m *mockMsgRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Message), args.Error(1)
}
func (m *mockMsgRepo) Update(ctx context.Context, msg *domain.Message) error {
	return m.Called(ctx, msg).Error(0)
}
func (m *mockMsgRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockMsgRepo) ListByConversation(ctx context.Context, convID uuid.UUID, limit, offset int) ([]*domain.Message, error) {
	args := m.Called(ctx, convID, limit, offset)
	return args.Get(0).([]*domain.Message), args.Error(1)
}
func (m *mockMsgRepo) Search(ctx context.Context, convID uuid.UUID, query string, limit, offset int) ([]*domain.Message, error) {
	args := m.Called(ctx, convID, query, limit, offset)
	return args.Get(0).([]*domain.Message), args.Error(1)
}
func (m *mockMsgRepo) SetPinned(ctx context.Context, id uuid.UUID, pinned bool) error {
	return m.Called(ctx, id, pinned).Error(0)
}

type mockReactionRepo struct{ mock.Mock }

func (m *mockReactionRepo) Add(ctx context.Context, r *domain.MessageReaction) error {
	return m.Called(ctx, r).Error(0)
}
func (m *mockReactionRepo) Remove(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	return m.Called(ctx, messageID, userID, emoji).Error(0)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func newMsgUC(convRepo *mockConvRepo, partRepo *mockPartRepo, msgRepo *mockMsgRepo, reactRepo *mockReactionRepo, pub *mockPublisher, cache *mockCache) *usecase.MessageUseCase {
	return usecase.NewMessageUseCase(convRepo, partRepo, msgRepo, reactRepo, pub, cache, &mockTransactor{})
}

func makeMessage(convID, senderID uuid.UUID) *domain.Message {
	return &domain.Message{
		ID:             uuid.New(),
		ConversationID: convID,
		SenderID:       senderID,
		Text:           "hello world",
		CreatedAt:      time.Now(),
	}
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestSendMessage_Success(t *testing.T) {
	convRepo := &mockConvRepo{}
	partRepo := &mockPartRepo{}
	msgRepo := &mockMsgRepo{}
	pub := &mockPublisher{}
	cache := &mockCache{}

	convID, senderID := uuid.New(), uuid.New()

	partRepo.On("IsParticipant", mock.Anything, convID, senderID).Return(true, nil)
	msgRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Message")).Return(nil)
	convRepo.On("UpdateLastMessageAt", mock.Anything, convID, mock.Anything).Return(nil)
	partRepo.On("IncrUnreadExceptSender", mock.Anything, convID, senderID).Return(nil)
	pub.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	partRepo.On("ListParticipants", mock.Anything, convID).Return([]*domain.Participant{}, nil)

	uc := newMsgUC(convRepo, partRepo, msgRepo, &mockReactionRepo{}, pub, cache)
	m, err := uc.SendMessage(context.Background(), usecase.SendMessageInput{
		ConvID:   convID,
		SenderID: senderID,
		Text:     "hello",
	})

	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, "hello", m.Text)
}

func TestSendMessage_NotParticipant(t *testing.T) {
	partRepo := &mockPartRepo{}
	convID, senderID := uuid.New(), uuid.New()
	partRepo.On("IsParticipant", mock.Anything, convID, senderID).Return(false, nil)

	uc := newMsgUC(&mockConvRepo{}, partRepo, &mockMsgRepo{}, &mockReactionRepo{}, &mockPublisher{}, &mockCache{})
	_, err := uc.SendMessage(context.Background(), usecase.SendMessageInput{ConvID: convID, SenderID: senderID, Text: "hi"})
	assert.ErrorIs(t, err, domain.ErrNotParticipant)
}

func TestSendMessage_Empty(t *testing.T) {
	partRepo := &mockPartRepo{}
	convID, senderID := uuid.New(), uuid.New()
	partRepo.On("IsParticipant", mock.Anything, convID, senderID).Return(true, nil)

	uc := newMsgUC(&mockConvRepo{}, partRepo, &mockMsgRepo{}, &mockReactionRepo{}, &mockPublisher{}, &mockCache{})
	_, err := uc.SendMessage(context.Background(), usecase.SendMessageInput{ConvID: convID, SenderID: senderID, Text: "   "})
	assert.ErrorIs(t, err, domain.ErrMessageEmpty)
}

func TestSendMessage_TooLong(t *testing.T) {
	partRepo := &mockPartRepo{}
	convID, senderID := uuid.New(), uuid.New()
	partRepo.On("IsParticipant", mock.Anything, convID, senderID).Return(true, nil)

	uc := newMsgUC(&mockConvRepo{}, partRepo, &mockMsgRepo{}, &mockReactionRepo{}, &mockPublisher{}, &mockCache{})
	long := make([]byte, domain.MaxMessageLen+1)
	_, err := uc.SendMessage(context.Background(), usecase.SendMessageInput{
		ConvID: convID, SenderID: senderID, Text: string(long),
	})
	assert.ErrorIs(t, err, domain.ErrMessageTooLong)
}

func TestSendMessage_PublishError_Ignored(t *testing.T) {
	convRepo := &mockConvRepo{}
	partRepo := &mockPartRepo{}
	msgRepo := &mockMsgRepo{}
	pub := &mockPublisher{}
	cache := &mockCache{}

	convID, senderID := uuid.New(), uuid.New()

	partRepo.On("IsParticipant", mock.Anything, convID, senderID).Return(true, nil)
	msgRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	convRepo.On("UpdateLastMessageAt", mock.Anything, convID, mock.Anything).Return(nil)
	partRepo.On("IncrUnreadExceptSender", mock.Anything, convID, senderID).Return(nil)
	pub.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
	partRepo.On("ListParticipants", mock.Anything, convID).Return([]*domain.Participant{}, nil)

	uc := newMsgUC(convRepo, partRepo, msgRepo, &mockReactionRepo{}, pub, cache)
	m, err := uc.SendMessage(context.Background(), usecase.SendMessageInput{ConvID: convID, SenderID: senderID, Text: "hi"})
	require.NoError(t, err)
	require.NotNil(t, m)
}

func TestEditMessage_Success(t *testing.T) {
	msgRepo := &mockMsgRepo{}
	convID, senderID := uuid.New(), uuid.New()
	msg := makeMessage(convID, senderID)

	msgRepo.On("GetByID", mock.Anything, msg.ID).Return(msg, nil)
	msgRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Message")).Return(nil)

	uc := newMsgUC(&mockConvRepo{}, &mockPartRepo{}, msgRepo, &mockReactionRepo{}, &mockPublisher{}, &mockCache{})
	updated, err := uc.EditMessage(context.Background(), msg.ID, senderID, "updated text")
	require.NoError(t, err)
	assert.Equal(t, "updated text", updated.Text)
	assert.NotNil(t, updated.EditedAt)
}

func TestEditMessage_NotOwner(t *testing.T) {
	msgRepo := &mockMsgRepo{}
	msg := makeMessage(uuid.New(), uuid.New())
	msgRepo.On("GetByID", mock.Anything, msg.ID).Return(msg, nil)

	uc := newMsgUC(&mockConvRepo{}, &mockPartRepo{}, msgRepo, &mockReactionRepo{}, &mockPublisher{}, &mockCache{})
	_, err := uc.EditMessage(context.Background(), msg.ID, uuid.New(), "hacked")
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestEditMessage_NotFound(t *testing.T) {
	msgRepo := &mockMsgRepo{}
	msgID := uuid.New()
	msgRepo.On("GetByID", mock.Anything, msgID).Return(nil, domain.ErrMessageNotFound)

	uc := newMsgUC(&mockConvRepo{}, &mockPartRepo{}, msgRepo, &mockReactionRepo{}, &mockPublisher{}, &mockCache{})
	_, err := uc.EditMessage(context.Background(), msgID, uuid.New(), "text")
	assert.ErrorIs(t, err, domain.ErrMessageNotFound)
}

func TestDeleteMessage_Success(t *testing.T) {
	msgRepo := &mockMsgRepo{}
	convID, senderID := uuid.New(), uuid.New()
	msg := makeMessage(convID, senderID)
	msgRepo.On("GetByID", mock.Anything, msg.ID).Return(msg, nil)
	msgRepo.On("SoftDelete", mock.Anything, msg.ID).Return(nil)

	uc := newMsgUC(&mockConvRepo{}, &mockPartRepo{}, msgRepo, &mockReactionRepo{}, &mockPublisher{}, &mockCache{})
	err := uc.DeleteMessage(context.Background(), msg.ID, senderID)
	require.NoError(t, err)
}

func TestDeleteMessage_NotOwner(t *testing.T) {
	msgRepo := &mockMsgRepo{}
	msg := makeMessage(uuid.New(), uuid.New())
	msgRepo.On("GetByID", mock.Anything, msg.ID).Return(msg, nil)

	uc := newMsgUC(&mockConvRepo{}, &mockPartRepo{}, msgRepo, &mockReactionRepo{}, &mockPublisher{}, &mockCache{})
	err := uc.DeleteMessage(context.Background(), msg.ID, uuid.New())
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestListMessages_NotParticipant(t *testing.T) {
	partRepo := &mockPartRepo{}
	convID, userID := uuid.New(), uuid.New()
	partRepo.On("IsParticipant", mock.Anything, convID, userID).Return(false, nil)

	uc := newMsgUC(&mockConvRepo{}, partRepo, &mockMsgRepo{}, &mockReactionRepo{}, &mockPublisher{}, &mockCache{})
	_, err := uc.ListMessages(context.Background(), convID, userID, 20, 0)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestListMessages_Success(t *testing.T) {
	partRepo := &mockPartRepo{}
	msgRepo := &mockMsgRepo{}

	convID, userID := uuid.New(), uuid.New()
	msgs := []*domain.Message{makeMessage(convID, userID), makeMessage(convID, uuid.New())}

	partRepo.On("IsParticipant", mock.Anything, convID, userID).Return(true, nil)
	msgRepo.On("ListByConversation", mock.Anything, convID, 20, 0).Return(msgs, nil)

	uc := newMsgUC(&mockConvRepo{}, partRepo, msgRepo, &mockReactionRepo{}, &mockPublisher{}, &mockCache{})
	got, err := uc.ListMessages(context.Background(), convID, userID, 0, 0)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestPinMessage_NotParticipant(t *testing.T) {
	msgRepo := &mockMsgRepo{}
	partRepo := &mockPartRepo{}

	convID, userID := uuid.New(), uuid.New()
	msg := makeMessage(convID, uuid.New())
	msgRepo.On("GetByID", mock.Anything, msg.ID).Return(msg, nil)
	partRepo.On("IsParticipant", mock.Anything, convID, userID).Return(false, nil)

	uc := newMsgUC(&mockConvRepo{}, partRepo, msgRepo, &mockReactionRepo{}, &mockPublisher{}, &mockCache{})
	err := uc.PinMessage(context.Background(), msg.ID, userID, true)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestPinMessage_Success(t *testing.T) {
	msgRepo := &mockMsgRepo{}
	partRepo := &mockPartRepo{}

	convID, userID := uuid.New(), uuid.New()
	msg := makeMessage(convID, uuid.New())
	msgRepo.On("GetByID", mock.Anything, msg.ID).Return(msg, nil)
	partRepo.On("IsParticipant", mock.Anything, convID, userID).Return(true, nil)
	msgRepo.On("SetPinned", mock.Anything, msg.ID, true).Return(nil)

	uc := newMsgUC(&mockConvRepo{}, partRepo, msgRepo, &mockReactionRepo{}, &mockPublisher{}, &mockCache{})
	err := uc.PinMessage(context.Background(), msg.ID, userID, true)
	require.NoError(t, err)
}

func TestAddReaction_Success(t *testing.T) {
	msgRepo := &mockMsgRepo{}
	partRepo := &mockPartRepo{}
	reactRepo := &mockReactionRepo{}

	convID, userID := uuid.New(), uuid.New()
	msg := makeMessage(convID, uuid.New())
	msgRepo.On("GetByID", mock.Anything, msg.ID).Return(msg, nil)
	partRepo.On("IsParticipant", mock.Anything, convID, userID).Return(true, nil)
	reactRepo.On("Add", mock.Anything, mock.AnythingOfType("*domain.MessageReaction")).Return(nil)

	uc := newMsgUC(&mockConvRepo{}, partRepo, msgRepo, reactRepo, &mockPublisher{}, &mockCache{})
	err := uc.AddReaction(context.Background(), msg.ID, userID, "👍")
	require.NoError(t, err)
}

func TestRemoveReaction_Success(t *testing.T) {
	msgRepo := &mockMsgRepo{}
	partRepo := &mockPartRepo{}
	reactRepo := &mockReactionRepo{}

	convID, userID := uuid.New(), uuid.New()
	msg := makeMessage(convID, uuid.New())
	msgRepo.On("GetByID", mock.Anything, msg.ID).Return(msg, nil)
	partRepo.On("IsParticipant", mock.Anything, convID, userID).Return(true, nil)
	reactRepo.On("Remove", mock.Anything, msg.ID, userID, "👍").Return(nil)

	uc := newMsgUC(&mockConvRepo{}, partRepo, msgRepo, reactRepo, &mockPublisher{}, &mockCache{})
	err := uc.RemoveReaction(context.Background(), msg.ID, userID, "👍")
	require.NoError(t, err)
}
