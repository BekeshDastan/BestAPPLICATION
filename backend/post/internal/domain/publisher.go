package domain

import "context"

const (
	EventPostCreated   = "post.created"
	EventPostDeleted   = "post.deleted"
	EventPostLiked     = "post.liked"
	EventPostUnliked   = "post.unliked"
	EventPostCommented = "post.commented"
)

type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}
