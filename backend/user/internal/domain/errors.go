package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("email already taken")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrAccountDeleted     = errors.New("account deleted")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenInvalid       = errors.New("token invalid")
	ErrTokenRevoked       = errors.New("token revoked")
	ErrForbidden          = errors.New("forbidden")
	ErrSelfFollow         = errors.New("cannot follow yourself")
	ErrAlreadyFollowing   = errors.New("already following")
	ErrNotFollowing       = errors.New("not following")
	ErrAlreadyBlocked     = errors.New("already blocked")
	ErrNotBlocked         = errors.New("not blocked")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrInvalidUsername    = errors.New("invalid username: 3-32 chars, alphanumeric and underscore only")
	ErrWeakPassword       = errors.New("password must be at least 8 characters")
)
