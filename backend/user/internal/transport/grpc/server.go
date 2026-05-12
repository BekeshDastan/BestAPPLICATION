// gRPC server: aggregates handlers, registers service, runs with graceful shutdown.
package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os/signal"
	"syscall"

	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// UserServer implements userv1.UserServiceServer by composing the three handler types.
type UserServer struct {
	userv1.UnimplementedUserServiceServer
	auth    *AuthHandler
	profile *ProfileHandler
	follow  *FollowHandler
}

func NewUserServer(auth *AuthHandler, profile *ProfileHandler, follow *FollowHandler) *UserServer {
	return &UserServer{auth: auth, profile: profile, follow: follow}
}

// ── Auth delegation ────────────────────────────────────────────────────────

func (s *UserServer) Register(ctx context.Context, r *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	return s.auth.Register(ctx, r)
}
func (s *UserServer) Login(ctx context.Context, r *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	return s.auth.Login(ctx, r)
}
func (s *UserServer) Logout(ctx context.Context, r *userv1.LogoutRequest) (*userv1.LogoutResponse, error) {
	return s.auth.Logout(ctx, r)
}
func (s *UserServer) RefreshToken(ctx context.Context, r *userv1.RefreshTokenRequest) (*userv1.RefreshTokenResponse, error) {
	return s.auth.RefreshToken(ctx, r)
}
func (s *UserServer) VerifyEmail(ctx context.Context, r *userv1.VerifyEmailRequest) (*userv1.VerifyEmailResponse, error) {
	return s.auth.VerifyEmail(ctx, r)
}
func (s *UserServer) ResendVerification(ctx context.Context, r *userv1.ResendVerificationRequest) (*userv1.ResendVerificationResponse, error) {
	return s.auth.ResendVerification(ctx, r)
}
func (s *UserServer) ForgotPassword(ctx context.Context, r *userv1.ForgotPasswordRequest) (*userv1.ForgotPasswordResponse, error) {
	return s.auth.ForgotPassword(ctx, r)
}
func (s *UserServer) ResetPassword(ctx context.Context, r *userv1.ResetPasswordRequest) (*userv1.ResetPasswordResponse, error) {
	return s.auth.ResetPassword(ctx, r)
}
func (s *UserServer) ChangePassword(ctx context.Context, r *userv1.ChangePasswordRequest) (*userv1.ChangePasswordResponse, error) {
	return s.auth.ChangePassword(ctx, r)
}
func (s *UserServer) ValidateToken(ctx context.Context, r *userv1.ValidateTokenRequest) (*userv1.ValidateTokenResponse, error) {
	return s.auth.ValidateToken(ctx, r)
}

// ── Profile delegation ─────────────────────────────────────────────────────

func (s *UserServer) GetProfile(ctx context.Context, r *userv1.GetProfileRequest) (*userv1.GetProfileResponse, error) {
	return s.profile.GetProfile(ctx, r)
}
func (s *UserServer) GetProfileByUsername(ctx context.Context, r *userv1.GetProfileByUsernameRequest) (*userv1.GetProfileResponse, error) {
	return s.profile.GetProfileByUsername(ctx, r)
}
func (s *UserServer) UpdateProfile(ctx context.Context, r *userv1.UpdateProfileRequest) (*userv1.UpdateProfileResponse, error) {
	return s.profile.UpdateProfile(ctx, r)
}
func (s *UserServer) UpdateAvatar(ctx context.Context, r *userv1.UpdateAvatarRequest) (*userv1.UpdateAvatarResponse, error) {
	return s.profile.UpdateAvatar(ctx, r)
}
func (s *UserServer) DeleteAccount(ctx context.Context, r *userv1.DeleteAccountRequest) (*userv1.DeleteAccountResponse, error) {
	return s.profile.DeleteAccount(ctx, r)
}
func (s *UserServer) SearchUsers(ctx context.Context, r *userv1.SearchUsersRequest) (*userv1.SearchUsersResponse, error) {
	return s.profile.SearchUsers(ctx, r)
}

// ── Follow delegation ──────────────────────────────────────────────────────

func (s *UserServer) Follow(ctx context.Context, r *userv1.FollowRequest) (*userv1.FollowResponse, error) {
	return s.follow.Follow(ctx, r)
}
func (s *UserServer) Unfollow(ctx context.Context, r *userv1.UnfollowRequest) (*userv1.UnfollowResponse, error) {
	return s.follow.Unfollow(ctx, r)
}
func (s *UserServer) ListFollowers(ctx context.Context, r *userv1.ListFollowersRequest) (*userv1.ListFollowersResponse, error) {
	return s.follow.ListFollowers(ctx, r)
}
func (s *UserServer) ListFollowing(ctx context.Context, r *userv1.ListFollowingRequest) (*userv1.ListFollowingResponse, error) {
	return s.follow.ListFollowing(ctx, r)
}
func (s *UserServer) IsFollowing(ctx context.Context, r *userv1.IsFollowingRequest) (*userv1.IsFollowingResponse, error) {
	return s.follow.IsFollowing(ctx, r)
}
func (s *UserServer) BlockUser(ctx context.Context, r *userv1.BlockUserRequest) (*userv1.BlockUserResponse, error) {
	return s.follow.BlockUser(ctx, r)
}
func (s *UserServer) UnblockUser(ctx context.Context, r *userv1.UnblockUserRequest) (*userv1.UnblockUserResponse, error) {
	return s.follow.UnblockUser(ctx, r)
}

// ── Run ────────────────────────────────────────────────────────────────────

// Run starts the gRPC listener and blocks until SIGINT/SIGTERM.
func Run(port string, srv *UserServer) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen :%s: %w", port, err)
	}

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(recoveryInterceptor),
	)
	userv1.RegisterUserServiceServer(grpcSrv, srv)
	reflection.Register(grpcSrv)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("gRPC server started", "port", port)
		if err := grpcSrv.Serve(lis); err != nil {
			slog.Error("gRPC serve error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down gRPC server")
	grpcSrv.GracefulStop()
	return nil
}

// recoveryInterceptor catches panics in handlers and returns Internal error.
func recoveryInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if p := recover(); p != nil {
			slog.Error("panic in gRPC handler", "panic", p)
			err = fmt.Errorf("internal server error")
		}
	}()
	return handler(ctx, req)
}
