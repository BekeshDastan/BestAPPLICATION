package domain

import "context"

// Event subjects published to NATS JetStream stream "SOCIAL".
const (
	EventUserRegistered         = "user.registered"
	EventUserFollowed           = "user.followed"
	EventUserUnfollowed         = "user.unfollowed"
	EventUserDeleted            = "user.deleted"
	EventPasswordResetRequested = "user.password_reset_requested"
)

type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}
