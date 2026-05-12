// gRPC follow handler: Follow, Unfollow, ListFollowers, ListFollowing,
// IsFollowing, BlockUser, UnblockUser.
package grpc

import (
	"context"

	"github.com/bekesh/social/backend/user/internal/usecase"
	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"github.com/google/uuid"
)

type FollowHandler struct {
	follow *usecase.FollowUseCase
}

func NewFollowHandler(follow *usecase.FollowUseCase) *FollowHandler {
	return &FollowHandler{follow: follow}
}

func (h *FollowHandler) Follow(ctx context.Context, req *userv1.FollowRequest) (*userv1.FollowResponse, error) {
	followerID, err := uuid.Parse(req.FollowerId)
	if err != nil {
		return nil, invalidArg("invalid follower_id")
	}
	followeeID, err := uuid.Parse(req.FolloweeId)
	if err != nil {
		return nil, invalidArg("invalid followee_id")
	}
	if err = h.follow.Follow(ctx, followerID, followeeID); err != nil {
		return nil, domainErr(err)
	}
	return &userv1.FollowResponse{}, nil
}

func (h *FollowHandler) Unfollow(ctx context.Context, req *userv1.UnfollowRequest) (*userv1.UnfollowResponse, error) {
	followerID, err := uuid.Parse(req.FollowerId)
	if err != nil {
		return nil, invalidArg("invalid follower_id")
	}
	followeeID, err := uuid.Parse(req.FolloweeId)
	if err != nil {
		return nil, invalidArg("invalid followee_id")
	}
	if err = h.follow.Unfollow(ctx, followerID, followeeID); err != nil {
		return nil, domainErr(err)
	}
	return &userv1.UnfollowResponse{}, nil
}

func (h *FollowHandler) ListFollowers(ctx context.Context, req *userv1.ListFollowersRequest) (*userv1.ListFollowersResponse, error) {
	id, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	users, err := h.follow.ListFollowers(ctx, id, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.ListFollowersResponse{Users: toProtoUsers(users)}, nil
}

func (h *FollowHandler) ListFollowing(ctx context.Context, req *userv1.ListFollowingRequest) (*userv1.ListFollowingResponse, error) {
	id, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	users, err := h.follow.ListFollowing(ctx, id, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.ListFollowingResponse{Users: toProtoUsers(users)}, nil
}

func (h *FollowHandler) IsFollowing(ctx context.Context, req *userv1.IsFollowingRequest) (*userv1.IsFollowingResponse, error) {
	followerID, err := uuid.Parse(req.FollowerId)
	if err != nil {
		return nil, invalidArg("invalid follower_id")
	}
	followeeID, err := uuid.Parse(req.FolloweeId)
	if err != nil {
		return nil, invalidArg("invalid followee_id")
	}
	ok, err := h.follow.IsFollowing(ctx, followerID, followeeID)
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.IsFollowingResponse{IsFollowing: ok}, nil
}

func (h *FollowHandler) BlockUser(ctx context.Context, req *userv1.BlockUserRequest) (*userv1.BlockUserResponse, error) {
	blockerID, err := uuid.Parse(req.BlockerId)
	if err != nil {
		return nil, invalidArg("invalid blocker_id")
	}
	blockedID, err := uuid.Parse(req.BlockedId)
	if err != nil {
		return nil, invalidArg("invalid blocked_id")
	}
	if err = h.follow.BlockUser(ctx, blockerID, blockedID); err != nil {
		return nil, domainErr(err)
	}
	return &userv1.BlockUserResponse{}, nil
}

func (h *FollowHandler) UnblockUser(ctx context.Context, req *userv1.UnblockUserRequest) (*userv1.UnblockUserResponse, error) {
	blockerID, err := uuid.Parse(req.BlockerId)
	if err != nil {
		return nil, invalidArg("invalid blocker_id")
	}
	blockedID, err := uuid.Parse(req.BlockedId)
	if err != nil {
		return nil, invalidArg("invalid blocked_id")
	}
	if err = h.follow.UnblockUser(ctx, blockerID, blockedID); err != nil {
		return nil, domainErr(err)
	}
	return &userv1.UnblockUserResponse{}, nil
}
