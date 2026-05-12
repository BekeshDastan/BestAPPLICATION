package domain

import "context"

const (
	EventStoryCreated = "story.created"
	EventStoryViewed  = "story.viewed"
)

type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}
