package domain

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrPostNotFound    = errors.New("post not found")
	ErrCommentNotFound = errors.New("comment not found")
	ErrAlreadyLiked    = errors.New("already liked")
	ErrNotLiked        = errors.New("not liked")
	ErrForbidden       = errors.New("forbidden")
	ErrCaptionTooLong  = errors.New("caption too long: max 2200 characters")
	ErrEmptyMedia      = errors.New("post must have at least one media item")
	ErrTooManyMedia    = errors.New("post can have at most 10 media items")
	ErrCommentEmpty    = errors.New("comment body cannot be empty")
	ErrCommentTooLong  = errors.New("comment too long: max 1000 characters")
	ErrAlreadySaved    = errors.New("post already saved")
	ErrNotSaved        = errors.New("post not saved")
)
