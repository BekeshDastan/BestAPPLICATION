package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/bekesh/social/backend/user/internal/domain"
	"github.com/google/uuid"
)

type FollowUseCase struct {
	users     domain.UserRepository
	follows   domain.FollowRepository
	publisher domain.EventPublisher
}

func NewFollowUseCase(
	users domain.UserRepository,
	follows domain.FollowRepository,
	publisher domain.EventPublisher,
) *FollowUseCase {
	return &FollowUseCase{users: users, follows: follows, publisher: publisher}
}

func (uc *FollowUseCase) Follow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	if followerID == followeeID {
		return domain.ErrSelfFollow
	}

	// Target user must exist
	target, err := uc.users.GetByID(ctx, followeeID)
	if err != nil {
		return fmt.Errorf("get followee: %w", err)
	}
	if target.IsDeleted() {
		return domain.ErrNotFound
	}

	// Cannot follow if blocked
	if blocked, err := uc.follows.IsBlocked(ctx, followeeID, followerID); err == nil && blocked {
		return domain.ErrForbidden
	}

	already, err := uc.follows.IsFollowing(ctx, followerID, followeeID)
	if err != nil {
		return fmt.Errorf("check following: %w", err)
	}
	if already {
		return domain.ErrAlreadyFollowing
	}

	if err = uc.follows.Follow(ctx, followerID, followeeID); err != nil {
		return fmt.Errorf("follow: %w", err)
	}
	_ = uc.publisher.Publish(ctx, domain.EventUserFollowed, map[string]string{
		"user_id":     followeeID.String(), // notification recipient
		"actor_id":    followerID.String(), // who performed the action
		"follower_id": followerID.String(),
		"followee_id": followeeID.String(),
	})
	return nil
}

func (uc *FollowUseCase) Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	if followerID == followeeID {
		return domain.ErrSelfFollow
	}

	following, err := uc.follows.IsFollowing(ctx, followerID, followeeID)
	if err != nil {
		return fmt.Errorf("check following: %w", err)
	}
	if !following {
		return domain.ErrNotFollowing
	}

	if err = uc.follows.Unfollow(ctx, followerID, followeeID); err != nil {
		return fmt.Errorf("unfollow: %w", err)
	}
	_ = uc.publisher.Publish(ctx, domain.EventUserUnfollowed, map[string]string{
		"follower_id": followerID.String(),
		"followee_id": followeeID.String(),
	})
	return nil
}

func (uc *FollowUseCase) ListFollowers(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.User, error) {
	limit, offset = clamp(limit, offset)
	users, err := uc.follows.ListFollowers(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list followers: %w", err)
	}
	return users, nil
}

func (uc *FollowUseCase) ListFollowing(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.User, error) {
	limit, offset = clamp(limit, offset)
	users, err := uc.follows.ListFollowing(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list following: %w", err)
	}
	return users, nil
}

func (uc *FollowUseCase) IsFollowing(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error) {
	result, err := uc.follows.IsFollowing(ctx, followerID, followeeID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return false, fmt.Errorf("check following: %w", err)
	}
	return result, nil
}

func (uc *FollowUseCase) BlockUser(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	if blockerID == blockedID {
		return domain.ErrForbidden
	}

	target, err := uc.users.GetByID(ctx, blockedID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if target.IsDeleted() {
		return domain.ErrNotFound
	}

	already, err := uc.follows.IsBlocked(ctx, blockerID, blockedID)
	if err != nil {
		return fmt.Errorf("check blocked: %w", err)
	}
	if already {
		return domain.ErrAlreadyBlocked
	}

	// Unfollow both directions silently
	_ = uc.follows.Unfollow(ctx, blockerID, blockedID)
	_ = uc.follows.Unfollow(ctx, blockedID, blockerID)

	return uc.follows.Block(ctx, blockerID, blockedID)
}

func (uc *FollowUseCase) UnblockUser(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	blocked, err := uc.follows.IsBlocked(ctx, blockerID, blockedID)
	if err != nil {
		return fmt.Errorf("check blocked: %w", err)
	}
	if !blocked {
		return domain.ErrNotBlocked
	}
	return uc.follows.Unblock(ctx, blockerID, blockedID)
}

func clamp(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
