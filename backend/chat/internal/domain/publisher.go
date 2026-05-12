package domain

import "context"

const EventChatMessageSent = "chat.message.sent"

type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}
