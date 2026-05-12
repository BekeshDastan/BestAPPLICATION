package grpc

import (
	"github.com/bekesh/social/backend/user/internal/domain"
	userv1 "github.com/bekesh/social/gen/go/user/v1"
)

func toProtoUser(u *domain.User) *userv1.UserProfile {
	return &userv1.UserProfile{
		Id:         u.ID.String(),
		Username:   string(u.Username),
		Email:      string(u.Email),
		FullName:   u.FullName,
		Bio:        u.Bio,
		AvatarUrl:  u.AvatarURL,
		IsVerified: u.IsVerified,
		IsPrivate:  u.IsPrivate,
		CreatedAt:  u.CreatedAt.Unix(),
	}
}

func toProtoUsers(users []*domain.User) []*userv1.UserProfile {
	out := make([]*userv1.UserProfile, 0, len(users))
	for _, u := range users {
		out = append(out, toProtoUser(u))
	}
	return out
}
