package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/bekesh/social/backend/notification/internal/domain"
	"github.com/bekesh/social/backend/notification/internal/usecase"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type eventPayload struct {
	UserID             string `json:"user_id"`
	ActorID            string `json:"actor_id"`
	PostID             string `json:"post_id"`
	StoryID            string `json:"story_id"`
	MessageID          string `json:"message_id"`
	ChatID             string `json:"chat_id"`
	Email              string `json:"email"`
	Username           string `json:"username"`
	VerificationToken  string `json:"verification_token"`
	Token              string `json:"token"`
}

type NATSConsumer struct {
	js      nats.JetStreamContext
	notifUC *usecase.NotificationUseCase
	email   domain.EmailSender
	appURL  string
}

func NewNATSConsumer(nc *nats.Conn, notifUC *usecase.NotificationUseCase, emailSender domain.EmailSender, appURL string) (*NATSConsumer, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}
	return &NATSConsumer{js: js, notifUC: notifUC, email: emailSender, appURL: appURL}, nil
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
		{domain.EventPasswordResetRequested, c.handlePasswordResetRequested},
	}

	for _, s := range subjects {
		sub, err := c.js.Subscribe(s.subject, func(msg *nats.Msg) {
			s.handler(ctx, msg)
			_ = msg.Ack()
		}, nats.Durable("notification-"+strings.ReplaceAll(s.subject, ".", "_")), nats.AckExplicit())
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

func (c *NATSConsumer) sendEmail(ctx context.Context, to, subject, body string) {
	if err := c.email.Send(ctx, to, subject, body); err != nil {
		slog.Error("failed to send email", "to", to, "subject", subject, "err", err)
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
		Type:          domain.NotificationTypeWelcome,
		ReferenceID:   parseUUID(p.UserID),
		ReferenceType: "user",
		Message:       "Welcome to Social!",
	})

	if p.Email != "" && p.VerificationToken != "" {
		link := fmt.Sprintf("%s/verify-email?token=%s&email=%s", c.appURL, p.VerificationToken, url.QueryEscape(p.Email))
		body := fmt.Sprintf(`<h2>Welcome, %s!</h2><p>Please verify your email:</p><p><a href="%s">Verify Email</a></p><p>Link: %s</p>`, p.Username, link, link)
		c.sendEmail(ctx, p.Email, "Verify your Social account", body)
	}
}

func (c *NATSConsumer) handlePasswordResetRequested(ctx context.Context, msg *nats.Msg) {
	p := parsePayload(msg.Data)
	if p.Email == "" || p.Token == "" {
		return
	}
	link := fmt.Sprintf("%s/reset-password?token=%s", c.appURL, p.Token)
	body := fmt.Sprintf(`<h2>Password Reset</h2><p>Click the link below to reset your password:</p><p><a href="%s">Reset Password</a></p><p>Link: %s</p><p>This link expires in 1 hour.</p>`, link, link)
	c.sendEmail(ctx, p.Email, "Reset your Social password", body)
}
