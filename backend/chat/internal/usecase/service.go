package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bekesh/social/backend/chat/internal/domain"
	"github.com/google/uuid"
)

func clamp(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// ── ConversationUseCase ────────────────────────────────────────────────────

type ConversationUseCase struct {
	convs domain.ConversationRepository
	parts domain.ParticipantRepository
	pub   domain.EventPublisher
	cache domain.ChatCache
	tx    domain.Transactor
}

func NewConversationUseCase(
	convs domain.ConversationRepository,
	parts domain.ParticipantRepository,
	pub domain.EventPublisher,
	cache domain.ChatCache,
	tx domain.Transactor,
) *ConversationUseCase {
	return &ConversationUseCase{convs: convs, parts: parts, pub: pub, cache: cache, tx: tx}
}

func (uc *ConversationUseCase) CreateConversation(ctx context.Context, creatorID uuid.UUID, memberIDs []uuid.UUID) (*domain.Conversation, error) {
	now := time.Now()
	conv := &domain.Conversation{
		ID:        uuid.New(),
		Type:      domain.ConvTypeDirect,
		CreatedBy: creatorID,
		CreatedAt: now,
	}

	allMembers := append([]uuid.UUID{creatorID}, memberIDs...)
	participants := make([]*domain.Participant, 0, len(allMembers))
	for _, uid := range allMembers {
		role := domain.RoleMember
		if uid == creatorID {
			role = domain.RoleOwner
		}
		participants = append(participants, &domain.Participant{
			ConversationID: conv.ID,
			UserID:         uid,
			Role:           role,
			JoinedAt:       now,
		})
	}

	if err := uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.convs.Create(ctx, conv); err != nil {
			return fmt.Errorf("create conversation: %w", err)
		}
		for _, p := range participants {
			if err := uc.parts.Add(ctx, p); err != nil {
				return fmt.Errorf("add participant: %w", err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return conv, nil
}

func (uc *ConversationUseCase) CreateGroupChat(ctx context.Context, creatorID uuid.UUID, name string, memberIDs []uuid.UUID) (*domain.Conversation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrInvalidGroupName
	}
	now := time.Now()
	conv := &domain.Conversation{
		ID:        uuid.New(),
		Type:      domain.ConvTypeGroup,
		Name:      name,
		CreatedBy: creatorID,
		CreatedAt: now,
	}

	allMembers := append([]uuid.UUID{creatorID}, memberIDs...)
	participants := make([]*domain.Participant, 0, len(allMembers))
	for _, uid := range allMembers {
		role := domain.RoleMember
		if uid == creatorID {
			role = domain.RoleOwner
		}
		participants = append(participants, &domain.Participant{
			ConversationID: conv.ID,
			UserID:         uid,
			Role:           role,
			JoinedAt:       now,
		})
	}

	if err := uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.convs.Create(ctx, conv); err != nil {
			return fmt.Errorf("create group: %w", err)
		}
		for _, p := range participants {
			if err := uc.parts.Add(ctx, p); err != nil {
				return fmt.Errorf("add participant: %w", err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return conv, nil
}

func (uc *ConversationUseCase) GetConversation(ctx context.Context, convID, userID uuid.UUID) (*domain.Conversation, error) {
	ok, err := uc.parts.IsParticipant(ctx, convID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrForbidden
	}
	conv, err := uc.convs.GetByID(ctx, convID)
	if err != nil {
		return nil, err
	}
	return conv, nil
}

func (uc *ConversationUseCase) ListConversations(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Conversation, error) {
	limit, offset = clamp(limit, offset)
	return uc.convs.ListByUser(ctx, userID, limit, offset)
}

func (uc *ConversationUseCase) DeleteConversation(ctx context.Context, convID, userID uuid.UUID) error {
	conv, err := uc.convs.GetByID(ctx, convID)
	if err != nil {
		return err
	}
	if conv.CreatedBy != userID {
		return domain.ErrForbidden
	}
	return uc.convs.Delete(ctx, convID)
}

func (uc *ConversationUseCase) MarkConversationRead(ctx context.Context, convID, userID uuid.UUID) error {
	ok, err := uc.parts.IsParticipant(ctx, convID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrForbidden
	}
	if err = uc.parts.MarkRead(ctx, convID, userID); err != nil {
		return err
	}
	_ = uc.cache.DelUnread(ctx, userID)
	return nil
}

func (uc *ConversationUseCase) AddParticipant(ctx context.Context, convID, requesterID, newUserID uuid.UUID) error {
	conv, err := uc.convs.GetByID(ctx, convID)
	if err != nil {
		return err
	}
	if conv.Type != domain.ConvTypeGroup {
		return domain.ErrGroupRequired
	}
	p, err := uc.parts.GetParticipant(ctx, convID, requesterID)
	if err != nil {
		return domain.ErrForbidden
	}
	if p.Role == domain.RoleMember {
		return domain.ErrForbidden
	}
	already, err := uc.parts.IsParticipant(ctx, convID, newUserID)
	if err != nil {
		return err
	}
	if already {
		return domain.ErrAlreadyParticipant
	}
	return uc.parts.Add(ctx, &domain.Participant{
		ConversationID: convID,
		UserID:         newUserID,
		Role:           domain.RoleMember,
		JoinedAt:       time.Now(),
	})
}

func (uc *ConversationUseCase) RemoveParticipant(ctx context.Context, convID, requesterID, targetUserID uuid.UUID) error {
	conv, err := uc.convs.GetByID(ctx, convID)
	if err != nil {
		return err
	}
	if conv.Type != domain.ConvTypeGroup {
		return domain.ErrGroupRequired
	}
	p, err := uc.parts.GetParticipant(ctx, convID, requesterID)
	if err != nil {
		return domain.ErrForbidden
	}
	if p.Role == domain.RoleMember {
		return domain.ErrForbidden
	}
	return uc.parts.Remove(ctx, convID, targetUserID)
}

func (uc *ConversationUseCase) LeaveGroup(ctx context.Context, convID, userID uuid.UUID) error {
	conv, err := uc.convs.GetByID(ctx, convID)
	if err != nil {
		return err
	}
	if conv.Type != domain.ConvTypeGroup {
		return domain.ErrGroupRequired
	}
	ok, err := uc.parts.IsParticipant(ctx, convID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrNotParticipant
	}
	return uc.parts.Remove(ctx, convID, userID)
}

func (uc *ConversationUseCase) UpdateGroupInfo(ctx context.Context, convID, requesterID uuid.UUID, name, avatarURL string) error {
	conv, err := uc.convs.GetByID(ctx, convID)
	if err != nil {
		return err
	}
	if conv.Type != domain.ConvTypeGroup {
		return domain.ErrGroupRequired
	}
	p, err := uc.parts.GetParticipant(ctx, convID, requesterID)
	if err != nil {
		return domain.ErrForbidden
	}
	if p.Role == domain.RoleMember {
		return domain.ErrForbidden
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.ErrInvalidGroupName
	}
	return uc.convs.UpdateInfo(ctx, convID, name, avatarURL)
}

// ── MessageUseCase ─────────────────────────────────────────────────────────

type SendMessageInput struct {
	ConvID    uuid.UUID
	SenderID  uuid.UUID
	ReplyToID *uuid.UUID
	Text      string
	MediaURL  string
}

type MessageUseCase struct {
	convs  domain.ConversationRepository
	parts  domain.ParticipantRepository
	msgs   domain.MessageRepository
	reacts domain.ReactionRepository
	pub    domain.EventPublisher
	cache  domain.ChatCache
	tx     domain.Transactor
}

func NewMessageUseCase(
	convs domain.ConversationRepository,
	parts domain.ParticipantRepository,
	msgs domain.MessageRepository,
	reacts domain.ReactionRepository,
	pub domain.EventPublisher,
	cache domain.ChatCache,
	tx domain.Transactor,
) *MessageUseCase {
	return &MessageUseCase{convs: convs, parts: parts, msgs: msgs, reacts: reacts, pub: pub, cache: cache, tx: tx}
}

func (uc *MessageUseCase) SendMessage(ctx context.Context, in SendMessageInput) (*domain.Message, error) {
	ok, err := uc.parts.IsParticipant(ctx, in.ConvID, in.SenderID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrNotParticipant
	}
	if strings.TrimSpace(in.Text) == "" && in.MediaURL == "" {
		return nil, domain.ErrMessageEmpty
	}
	if len(in.Text) > domain.MaxMessageLen {
		return nil, domain.ErrMessageTooLong
	}

	now := time.Now()
	m := &domain.Message{
		ID:             uuid.New(),
		ConversationID: in.ConvID,
		SenderID:       in.SenderID,
		ReplyToID:      in.ReplyToID,
		Text:           strings.TrimSpace(in.Text),
		MediaURL:       in.MediaURL,
		CreatedAt:      now,
	}

	if err = uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.msgs.Create(ctx, m); err != nil {
			return fmt.Errorf("create message: %w", err)
		}
		if err := uc.convs.UpdateLastMessageAt(ctx, in.ConvID, now); err != nil {
			return fmt.Errorf("update last_message_at: %w", err)
		}
		return uc.parts.IncrUnreadExceptSender(ctx, in.ConvID, in.SenderID)
	}); err != nil {
		return nil, err
	}

	subject := domain.EventChatMessageSent + "." + in.ConvID.String()
	_ = uc.pub.Publish(ctx, subject, map[string]string{
		"message_id":      m.ID.String(),
		"conversation_id": in.ConvID.String(),
		"sender_id":       in.SenderID.String(),
		"text":            m.Text,
	})

	if participants, err := uc.parts.ListParticipants(ctx, in.ConvID); err == nil {
		for _, p := range participants {
			if p.UserID == in.SenderID {
				continue
			}
			_, _ = uc.cache.IncrUnread(ctx, p.UserID)
			// One notification event per recipient (user_id = recipient).
			_ = uc.pub.Publish(ctx, domain.EventChatMessageSent, map[string]string{
				"message_id":      m.ID.String(),
				"conversation_id": in.ConvID.String(),
				"chat_id":         in.ConvID.String(),
				"sender_id":       in.SenderID.String(),
				"user_id":         p.UserID.String(),     // notification recipient
				"actor_id":        in.SenderID.String(),  // who sent
				"text":            m.Text,
			})
		}
	}

	return m, nil
}

func (uc *MessageUseCase) EditMessage(ctx context.Context, msgID, senderID uuid.UUID, text string) (*domain.Message, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, domain.ErrMessageEmpty
	}
	if len(text) > domain.MaxMessageLen {
		return nil, domain.ErrMessageTooLong
	}
	m, err := uc.msgs.GetByID(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if m.IsDeleted() {
		return nil, domain.ErrMessageNotFound
	}
	if m.SenderID != senderID {
		return nil, domain.ErrForbidden
	}
	now := time.Now()
	m.Text = text
	m.EditedAt = &now
	if err = uc.msgs.Update(ctx, m); err != nil {
		return nil, fmt.Errorf("update message: %w", err)
	}
	return m, nil
}

func (uc *MessageUseCase) DeleteMessage(ctx context.Context, msgID, senderID uuid.UUID) error {
	m, err := uc.msgs.GetByID(ctx, msgID)
	if err != nil {
		return err
	}
	if m.IsDeleted() {
		return domain.ErrMessageNotFound
	}
	if m.SenderID != senderID {
		return domain.ErrForbidden
	}
	return uc.msgs.SoftDelete(ctx, msgID)
}

func (uc *MessageUseCase) ListMessages(ctx context.Context, convID, userID uuid.UUID, limit, offset int) ([]*domain.Message, error) {
	ok, err := uc.parts.IsParticipant(ctx, convID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrForbidden
	}
	limit, offset = clamp(limit, offset)
	messages, err := uc.msgs.ListByConversation(ctx, convID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Message, 0, len(messages))
	for _, m := range messages {
		if !m.IsDeleted() {
			out = append(out, m)
		}
	}
	return out, nil
}

func (uc *MessageUseCase) SearchMessages(ctx context.Context, convID, userID uuid.UUID, query string, limit, offset int) ([]*domain.Message, error) {
	ok, err := uc.parts.IsParticipant(ctx, convID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrForbidden
	}
	limit, offset = clamp(limit, offset)
	return uc.msgs.Search(ctx, convID, query, limit, offset)
}

func (uc *MessageUseCase) PinMessage(ctx context.Context, msgID, userID uuid.UUID, pinned bool) error {
	m, err := uc.msgs.GetByID(ctx, msgID)
	if err != nil {
		return err
	}
	if m.IsDeleted() {
		return domain.ErrMessageNotFound
	}
	ok, err := uc.parts.IsParticipant(ctx, m.ConversationID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrForbidden
	}
	return uc.msgs.SetPinned(ctx, msgID, pinned)
}

func (uc *MessageUseCase) AddReaction(ctx context.Context, msgID, userID uuid.UUID, emoji string) error {
	m, err := uc.msgs.GetByID(ctx, msgID)
	if err != nil {
		return err
	}
	if m.IsDeleted() {
		return domain.ErrMessageNotFound
	}
	ok, err := uc.parts.IsParticipant(ctx, m.ConversationID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrForbidden
	}
	return uc.reacts.Add(ctx, &domain.MessageReaction{
		MessageID: msgID,
		UserID:    userID,
		Emoji:     emoji,
	})
}

func (uc *MessageUseCase) RemoveReaction(ctx context.Context, msgID, userID uuid.UUID, emoji string) error {
	m, err := uc.msgs.GetByID(ctx, msgID)
	if err != nil {
		return err
	}
	if m.IsDeleted() {
		return domain.ErrMessageNotFound
	}
	ok, err := uc.parts.IsParticipant(ctx, m.ConversationID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrForbidden
	}
	return uc.reacts.Remove(ctx, msgID, userID, emoji)
}
