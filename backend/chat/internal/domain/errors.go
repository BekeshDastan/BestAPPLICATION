package domain

import "errors"

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound      = errors.New("message not found")
	ErrForbidden            = errors.New("forbidden")
	ErrNotParticipant       = errors.New("not a participant")
	ErrAlreadyParticipant   = errors.New("already a participant")
	ErrMessageEmpty         = errors.New("message text is empty")
	ErrMessageTooLong       = errors.New("message text is too long")
	ErrNotFound             = errors.New("not found")
	ErrDuplicateReaction    = errors.New("reaction already exists")
	ErrInvalidGroupName     = errors.New("group name is required")
	ErrGroupRequired        = errors.New("operation requires a group conversation")
)
