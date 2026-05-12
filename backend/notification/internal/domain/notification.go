package domain

import (
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	NotificationTypeLike      NotificationType = "like"
	NotificationTypeComment   NotificationType = "comment"
	NotificationTypeFollow    NotificationType = "follow"
	NotificationTypeMessage   NotificationType = "message"
	NotificationTypeStoryView NotificationType = "story_view"
	NotificationTypeMention   NotificationType = "mention"
)

type Notification struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	ActorID       uuid.UUID
	Type          NotificationType
	ReferenceID   uuid.UUID
	ReferenceType string
	Message       string
	IsRead        bool
	CreatedAt     time.Time
}

type NotificationPreference struct {
	UserID       uuid.UUID
	Type         NotificationType
	EmailEnabled bool
	PushEnabled  bool
}
