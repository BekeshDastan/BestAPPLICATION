// gRPC profile handler: GetProfile, GetProfileByUsername, UpdateProfile,
// UpdateAvatar, DeleteAccount, SearchUsers.
package grpc

import (
	"context"

	"github.com/bekesh/social/backend/user/internal/usecase"
	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"github.com/google/uuid"
)

type ProfileHandler struct {
	profile *usecase.ProfileUseCase
}

func NewProfileHandler(profile *usecase.ProfileUseCase) *ProfileHandler {
	return &ProfileHandler{profile: profile}
}

func (h *ProfileHandler) GetProfile(ctx context.Context, req *userv1.GetProfileRequest) (*userv1.GetProfileResponse, error) {
	id, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	user, err := h.profile.GetProfile(ctx, id)
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.GetProfileResponse{User: toProtoUser(user)}, nil
}

func (h *ProfileHandler) GetProfileByUsername(ctx context.Context, req *userv1.GetProfileByUsernameRequest) (*userv1.GetProfileResponse, error) {
	user, err := h.profile.GetProfileByUsername(ctx, req.Username)
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.GetProfileResponse{User: toProtoUser(user)}, nil
}

func (h *ProfileHandler) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.UpdateProfileResponse, error) {
	id, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	user, err := h.profile.UpdateProfile(ctx, id, usecase.UpdateProfileInput{
		FullName:  req.FullName,
		Bio:       req.Bio,
		IsPrivate: req.IsPrivate,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.UpdateProfileResponse{User: toProtoUser(user)}, nil
}

func (h *ProfileHandler) UpdateAvatar(ctx context.Context, req *userv1.UpdateAvatarRequest) (*userv1.UpdateAvatarResponse, error) {
	id, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	user, err := h.profile.UpdateAvatar(ctx, id, req.AvatarUrl)
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.UpdateAvatarResponse{User: toProtoUser(user)}, nil
}

func (h *ProfileHandler) DeleteAccount(ctx context.Context, req *userv1.DeleteAccountRequest) (*userv1.DeleteAccountResponse, error) {
	id, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.profile.DeleteAccount(ctx, id, req.Password); err != nil {
		return nil, domainErr(err)
	}
	return &userv1.DeleteAccountResponse{}, nil
}

func (h *ProfileHandler) SearchUsers(ctx context.Context, req *userv1.SearchUsersRequest) (*userv1.SearchUsersResponse, error) {
	users, err := h.profile.SearchUsers(ctx, req.Query, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.SearchUsersResponse{Users: toProtoUsers(users)}, nil
}
