package domain

import "context"

// Inbound events consumed from NATS (published by other services).
const (
	EventPostLiked     = "post.liked"
	EventPostCommented = "post.commented"
	EventUserFollowed  = "user.followed"
	EventUserRegistered = "user.registered"
	EventStoryViewed   = "story.viewed"
	EventMessageSent   = "chat.message.sent"
)

// Outbound events published by the notification service.
const EventNotificationCreated = "notification.created"

type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}
