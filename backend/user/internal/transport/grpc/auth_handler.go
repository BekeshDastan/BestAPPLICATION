// gRPC auth handler: Register, Login, Logout, RefreshToken, VerifyEmail,
// ResendVerification, ForgotPassword, ResetPassword, ChangePassword, ValidateToken.
package grpc

import (
	"context"

	"github.com/bekesh/social/backend/user/internal/usecase"
	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"github.com/google/uuid"
)

type AuthHandler struct {
	auth *usecase.AuthUseCase
}

func NewAuthHandler(auth *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	user, pair, err := h.auth.Register(ctx, req.Email, req.Username, req.Password, req.FullName)
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.RegisterResponse{
		User:   toProtoUser(user),
		Tokens: &userv1.TokenPair{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken},
	}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	user, pair, err := h.auth.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.LoginResponse{
		User:   toProtoUser(user),
		Tokens: &userv1.TokenPair{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken},
	}, nil
}

func (h *AuthHandler) Logout(ctx context.Context, req *userv1.LogoutRequest) (*userv1.LogoutResponse, error) {
	if err := h.auth.Logout(ctx, req.RefreshToken); err != nil {
		return nil, domainErr(err)
	}
	return &userv1.LogoutResponse{}, nil
}

func (h *AuthHandler) RefreshToken(ctx context.Context, req *userv1.RefreshTokenRequest) (*userv1.RefreshTokenResponse, error) {
	pair, err := h.auth.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.RefreshTokenResponse{
		Tokens: &userv1.TokenPair{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken},
	}, nil
}

func (h *AuthHandler) VerifyEmail(ctx context.Context, req *userv1.VerifyEmailRequest) (*userv1.VerifyEmailResponse, error) {
	if err := h.auth.VerifyEmail(ctx, req.Token); err != nil {
		return nil, domainErr(err)
	}
	return &userv1.VerifyEmailResponse{}, nil
}

func (h *AuthHandler) ResendVerification(ctx context.Context, req *userv1.ResendVerificationRequest) (*userv1.ResendVerificationResponse, error) {
	if err := h.auth.ResendVerification(ctx, req.Email); err != nil {
		return nil, domainErr(err)
	}
	return &userv1.ResendVerificationResponse{}, nil
}

func (h *AuthHandler) ForgotPassword(ctx context.Context, req *userv1.ForgotPasswordRequest) (*userv1.ForgotPasswordResponse, error) {
	if err := h.auth.ForgotPassword(ctx, req.Email); err != nil {
		return nil, domainErr(err)
	}
	return &userv1.ForgotPasswordResponse{}, nil
}

func (h *AuthHandler) ResetPassword(ctx context.Context, req *userv1.ResetPasswordRequest) (*userv1.ResetPasswordResponse, error) {
	if err := h.auth.ResetPassword(ctx, req.Token, req.NewPassword); err != nil {
		return nil, domainErr(err)
	}
	return &userv1.ResetPasswordResponse{}, nil
}

func (h *AuthHandler) ChangePassword(ctx context.Context, req *userv1.ChangePasswordRequest) (*userv1.ChangePasswordResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, invalidArg("invalid user_id")
	}
	if err = h.auth.ChangePassword(ctx, userID, req.OldPassword, req.NewPassword); err != nil {
		return nil, domainErr(err)
	}
	return &userv1.ChangePasswordResponse{}, nil
}

func (h *AuthHandler) ValidateToken(ctx context.Context, req *userv1.ValidateTokenRequest) (*userv1.ValidateTokenResponse, error) {
	claims, err := h.auth.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.ValidateTokenResponse{
		UserId:   claims.UserID.String(),
		Username: claims.Username,
		Email:    claims.Email,
	}, nil
}
