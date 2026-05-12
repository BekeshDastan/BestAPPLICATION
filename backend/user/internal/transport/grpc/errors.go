package grpc

import (
	"errors"

	"github.com/bekesh/social/backend/user/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func invalidArg(msg string) error {
	return status.Error(codes.InvalidArgument, msg)
}

func domainErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound),
		errors.Is(err, domain.ErrNotFollowing),
		errors.Is(err, domain.ErrNotBlocked):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, domain.ErrEmailTaken),
		errors.Is(err, domain.ErrUsernameTaken),
		errors.Is(err, domain.ErrAlreadyFollowing),
		errors.Is(err, domain.ErrAlreadyBlocked):
		return status.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrTokenExpired),
		errors.Is(err, domain.ErrTokenInvalid),
		errors.Is(err, domain.ErrTokenRevoked):
		return status.Error(codes.Unauthenticated, err.Error())

	case errors.Is(err, domain.ErrEmailNotVerified):
		return status.Error(codes.FailedPrecondition, err.Error())

	case errors.Is(err, domain.ErrAccountDeleted),
		errors.Is(err, domain.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())

	case errors.Is(err, domain.ErrSelfFollow),
		errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidUsername),
		errors.Is(err, domain.ErrWeakPassword):
		return status.Error(codes.InvalidArgument, err.Error())

	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
