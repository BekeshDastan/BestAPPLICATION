package domain

import "errors"

var (
	ErrStoryNotFound         = errors.New("story not found")
	ErrHighlightNotFound     = errors.New("highlight not found")
	ErrNotFound              = errors.New("not found")
	ErrForbidden             = errors.New("access denied")
	ErrStoryExpired          = errors.New("story has expired")
	ErrAlreadyViewed         = errors.New("story already viewed")
	ErrAlreadyReacted        = errors.New("already reacted")
	ErrReactionNotFound      = errors.New("reaction not found")
	ErrInvalidHighlightTitle = errors.New("highlight title cannot be empty")
	ErrMediaURLRequired      = errors.New("media_url is required")
	ErrInvalidMediaType      = errors.New("media_type must be image or video")
	ErrReplyTextEmpty        = errors.New("reply text cannot be empty")
)
