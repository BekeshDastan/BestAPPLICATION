package domain

import "errors"

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrPreferenceNotFound   = errors.New("preference not found")
	ErrForbidden            = errors.New("access denied")
	ErrInvalidType          = errors.New("invalid notification type")
	ErrUserIDRequired       = errors.New("user_id is required")
	ErrMessageRequired      = errors.New("message is required")
)
