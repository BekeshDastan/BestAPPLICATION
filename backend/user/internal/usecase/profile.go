package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bekesh/social/backend/user/internal/domain"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UpdateProfileInput struct {
	FullName  string
	Bio       string
	IsPrivate bool
}

type ProfileUseCase struct {
	users     domain.UserRepository
	cache     domain.UserCache
	publisher domain.EventPublisher
}

func NewProfileUseCase(
	users domain.UserRepository,
	cache domain.UserCache,
	publisher domain.EventPublisher,
) *ProfileUseCase {
	return &ProfileUseCase{users: users, cache: cache, publisher: publisher}
}

func (uc *ProfileUseCase) GetProfile(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if cached, err := uc.cache.GetProfile(ctx, id); err == nil {
		return cached, nil
	}
	user, err := uc.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user.IsDeleted() {
		return nil, domain.ErrNotFound
	}
	_ = uc.cache.SetProfile(ctx, user, 5*time.Minute)
	return user, nil
}

func (uc *ProfileUseCase) GetProfileByUsername(ctx context.Context, username string) (*domain.User, error) {
	user, err := uc.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user.IsDeleted() {
		return nil, domain.ErrNotFound
	}
	return user, nil
}

func (uc *ProfileUseCase) UpdateProfile(ctx context.Context, userID uuid.UUID, in UpdateProfileInput) (*domain.User, error) {
	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user.IsDeleted() {
		return nil, domain.ErrNotFound
	}
	user.FullName = in.FullName
	user.Bio = in.Bio
	user.IsPrivate = in.IsPrivate
	user.UpdatedAt = time.Now()

	if err = uc.users.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	_ = uc.cache.InvalidateProfile(ctx, userID)
	return user, nil
}

func (uc *ProfileUseCase) UpdateAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) (*domain.User, error) {
	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user.IsDeleted() {
		return nil, domain.ErrNotFound
	}
	user.AvatarURL = avatarURL
	user.UpdatedAt = time.Now()

	if err = uc.users.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	_ = uc.cache.InvalidateProfile(ctx, userID)
	return user, nil
}

func (uc *ProfileUseCase) DeleteAccount(ctx context.Context, userID uuid.UUID, rawPassword string) error {
	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(rawPassword)) != nil {
		return domain.ErrInvalidCredentials
	}
	return uc.deleteUser(ctx, userID)
}

// AdminDeleteUser removes a user without password verification.
// Authorisation must be enforced by the caller (gateway admin middleware).
func (uc *ProfileUseCase) AdminDeleteUser(ctx context.Context, userID uuid.UUID) error {
	if _, err := uc.users.GetByID(ctx, userID); err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	return uc.deleteUser(ctx, userID)
}

func (uc *ProfileUseCase) deleteUser(ctx context.Context, userID uuid.UUID) error {
	if err := uc.users.SoftDelete(ctx, userID); err != nil {
		return fmt.Errorf("soft delete: %w", err)
	}
	_ = uc.cache.InvalidateProfile(ctx, userID)
	_ = uc.publisher.Publish(ctx, domain.EventUserDeleted, map[string]string{
		"user_id": userID.String(),
	})
	return nil
}

func (uc *ProfileUseCase) SearchUsers(ctx context.Context, query string, limit, offset int) ([]*domain.User, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	users, err := uc.users.Search(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	result := make([]*domain.User, 0, len(users))
	for _, u := range users {
		if !u.IsDeleted() {
			result = append(result, u)
		}
	}
	return result, nil
}
