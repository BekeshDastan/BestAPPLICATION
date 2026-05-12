package messaging

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/bekesh/social/backend/notification/internal/domain"
	"github.com/bekesh/social/backend/notification/internal/usecase"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type eventPayload struct {
	UserID    string `json:"user_id"`
	ActorID   string `json:"actor_id"`
	PostID    string `json:"post_id"`
	StoryID   string `json:"story_id"`
	MessageID string `json:"message_id"`
	ChatID    string `json:"chat_id"`
}

type NATSConsumer struct {
	js      nats.JetStreamContext
	notifUC *usecase.NotificationUseCase
}

func NewNATSConsumer(nc *nats.Conn, notifUC *usecase.NotificationUseCase) (*NATSConsumer, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}
	return &NATSConsumer{js: js, notifUC: notifUC}, nil
}

func (c *NATSConsumer) Start(ctx context.Context) {
	subjects := []struct {
		subject string
		handler func(context.Context, *nats.Msg)
	}{
		{domain.EventPostLiked, c.handlePostLiked},
		{domain.EventPostCommented, c.handlePostCommented},
		{domain.EventUserFollowed, c.handleUserFollowed},
		{domain.EventStoryViewed, c.handleStoryViewed},
		{domain.EventMessageSent, c.handleMessageSent},
		{domain.EventUserRegistered, c.handleUserRegistered},
	}

	for _, s := range subjects {
		sub, err := c.js.Subscribe(s.subject, func(msg *nats.Msg) {
			s.handler(ctx, msg)
			_ = msg.Ack()
		}, nats.Durable("notification-"+s.subject), nats.AckExplicit())
		if err != nil {
			slog.Error("failed to subscribe", "subject", s.subject, "err", err)
			continue
		}
		slog.Info("subscribed to NATS subject", "subject", s.subject)
		go func(sub *nats.Subscription) {
			<-ctx.Done()
			_ = sub.Unsubscribe()
		}(sub)
	}
}

func parsePayload(data []byte) eventPayload {
	var p eventPayload
	_ = json.Unmarshal(data, &p)
	return p
}

func parseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

func (c *NATSConsumer) create(ctx context.Context, n *domain.Notification) {
	n.CreatedAt = time.Now()
	if _, err := c.notifUC.Create(ctx, n); err != nil {
		slog.Error("failed to create notification", "err", err, "type", n.Type)
	}
}

func (c *NATSConsumer) handlePostLiked(ctx context.Context, msg *nats.Msg) {
	p := parsePayload(msg.Data)
	c.create(ctx, &domain.Notification{
		UserID:        parseUUID(p.UserID),
		ActorID:       parseUUID(p.ActorID),
		Type:          domain.NotificationTypeLike,
		ReferenceID:   parseUUID(p.PostID),
		ReferenceType: "post",
		Message:       "liked your post",
	})
}

func (c *NATSConsumer) handlePostCommented(ctx context.Context, msg *nats.Msg) {
	p := parsePayload(msg.Data)
	c.create(ctx, &domain.Notification{
		UserID:        parseUUID(p.UserID),
		ActorID:       parseUUID(p.ActorID),
		Type:          domain.NotificationTypeComment,
		ReferenceID:   parseUUID(p.PostID),
		ReferenceType: "post",
		Message:       "commented on your post",
	})
}

func (c *NATSConsumer) handleUserFollowed(ctx context.Context, msg *nats.Msg) {
	p := parsePayload(msg.Data)
	c.create(ctx, &domain.Notification{
		UserID:        parseUUID(p.UserID),
		ActorID:       parseUUID(p.ActorID),
		Type:          domain.NotificationTypeFollow,
		ReferenceID:   parseUUID(p.ActorID),
		ReferenceType: "user",
		Message:       "started following you",
	})
}

func (c *NATSConsumer) handleStoryViewed(ctx context.Context, msg *nats.Msg) {
	p := parsePayload(msg.Data)
	c.create(ctx, &domain.Notification{
		UserID:        parseUUID(p.UserID),
		ActorID:       parseUUID(p.ActorID),
		Type:          domain.NotificationTypeStoryView,
		ReferenceID:   parseUUID(p.StoryID),
		ReferenceType: "story",
		Message:       "viewed your story",
	})
}

func (c *NATSConsumer) handleMessageSent(ctx context.Context, msg *nats.Msg) {
	p := parsePayload(msg.Data)
	c.create(ctx, &domain.Notification{
		UserID:        parseUUID(p.UserID),
		ActorID:       parseUUID(p.ActorID),
		Type:          domain.NotificationTypeMessage,
		ReferenceID:   parseUUID(p.ChatID),
		ReferenceType: "chat",
		Message:       "sent you a message",
	})
}

func (c *NATSConsumer) handleUserRegistered(ctx context.Context, msg *nats.Msg) {
	p := parsePayload(msg.Data)
	c.create(ctx, &domain.Notification{
		UserID:        parseUUID(p.UserID),
		ActorID:       parseUUID(p.UserID),
		Type:          domain.NotificationTypeFollow,
		ReferenceID:   parseUUID(p.UserID),
		ReferenceType: "user",
		Message:       "welcome to Social!",
	})
}
