package usecase

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/bekesh/social/backend/user/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type Claims struct {
	UserID   uuid.UUID
	Username string
	Email    string
}

type AuthUseCase struct {
	users     domain.UserRepository
	tokens    domain.TokenRepository
	cache     domain.UserCache
	publisher domain.EventPublisher
	tx        domain.Transactor
	privKey   *ecdsa.PrivateKey
	pubKey    *ecdsa.PublicKey
	accessTTL time.Duration
	refreshTTL time.Duration
}

func NewAuthUseCase(
	users domain.UserRepository,
	tokens domain.TokenRepository,
	cache domain.UserCache,
	publisher domain.EventPublisher,
	tx domain.Transactor,
	privKey *ecdsa.PrivateKey,
	pubKey *ecdsa.PublicKey,
	accessTTL, refreshTTL time.Duration,
) *AuthUseCase {
	return &AuthUseCase{
		users:      users,
		tokens:     tokens,
		cache:      cache,
		publisher:  publisher,
		tx:         tx,
		privKey:    privKey,
		pubKey:     pubKey,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (uc *AuthUseCase) Register(ctx context.Context, rawEmail, rawUsername, rawPassword, fullName string) (*domain.User, *TokenPair, error) {
	email, err := domain.NewEmail(rawEmail)
	if err != nil {
		return nil, nil, err
	}
	username, err := domain.NewUsername(rawUsername)
	if err != nil {
		return nil, nil, err
	}
	if _, err = domain.NewRawPassword(rawPassword); err != nil {
		return nil, nil, err
	}

	if _, err = uc.users.GetByEmail(ctx, string(email)); err == nil {
		return nil, nil, domain.ErrEmailTaken
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, nil, fmt.Errorf("check email: %w", err)
	}

	if _, err = uc.users.GetByUsername(ctx, string(username)); err == nil {
		return nil, nil, domain.ErrUsernameTaken
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, nil, fmt.Errorf("check username: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcryptCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		FullName:     fullName,
		IsVerified:   false,
		IsPrivate:    false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	evToken, err := generateSecureToken()
	if err != nil {
		return nil, nil, fmt.Errorf("generate verification token: %w", err)
	}
	ev := &domain.EmailVerification{
		Token:     evToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err = uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.users.Create(ctx, user); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		if err := uc.tokens.SaveEmailVerification(ctx, ev); err != nil {
			return fmt.Errorf("save email verification: %w", err)
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}

	pair, err := uc.issuePair(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	_ = uc.publisher.Publish(ctx, domain.EventUserRegistered, map[string]string{
		"user_id":            user.ID.String(),
		"email":              string(user.Email),
		"username":           string(user.Username),
		"verification_token": evToken,
	})

	return user, pair, nil
}

func (uc *AuthUseCase) Login(ctx context.Context, rawEmail, rawPassword string) (*domain.User, *TokenPair, error) {
	user, err := uc.users.GetByEmail(ctx, rawEmail)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, domain.ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("get user: %w", err)
	}

	if user.IsDeleted() {
		return nil, nil, domain.ErrAccountDeleted
	}
	if !user.IsVerified {
		return nil, nil, domain.ErrEmailNotVerified
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(rawPassword)); err != nil {
		return nil, nil, domain.ErrInvalidCredentials
	}

	pair, err := uc.issuePair(ctx, user)
	if err != nil {
		return nil, nil, err
	}
	return user, pair, nil
}

func (uc *AuthUseCase) Logout(ctx context.Context, rawRefreshToken string) error {
	tokenHash := hashToken(rawRefreshToken)
	rt, err := uc.tokens.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil // already gone
		}
		return fmt.Errorf("get refresh token: %w", err)
	}
	if err = uc.tokens.RevokeRefreshToken(ctx, rt.ID); err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	ttl := time.Until(rt.ExpiresAt)
	if ttl > 0 {
		_ = uc.cache.BlacklistToken(ctx, tokenHash, ttl)
	}
	return nil
}

func (uc *AuthUseCase) RefreshToken(ctx context.Context, rawRefreshToken string) (*TokenPair, error) {
	tokenHash := hashToken(rawRefreshToken)

	blacklisted, err := uc.cache.IsTokenBlacklisted(ctx, tokenHash)
	if err == nil && blacklisted {
		return nil, domain.ErrTokenRevoked
	}

	rt, err := uc.tokens.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrTokenInvalid
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	if !rt.IsValid() {
		if rt.IsExpired() {
			return nil, domain.ErrTokenExpired
		}
		return nil, domain.ErrTokenRevoked
	}

	user, err := uc.users.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user.IsDeleted() {
		return nil, domain.ErrAccountDeleted
	}

	// Rotate: revoke old, issue new
	if err = uc.tokens.RevokeRefreshToken(ctx, rt.ID); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}
	_ = uc.cache.BlacklistToken(ctx, tokenHash, time.Until(rt.ExpiresAt))

	return uc.issuePair(ctx, user)
}

func (uc *AuthUseCase) VerifyEmail(ctx context.Context, token string) error {
	ev, err := uc.tokens.GetEmailVerification(ctx, token)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrTokenInvalid
		}
		return fmt.Errorf("get email verification: %w", err)
	}
	if ev.IsExpired() {
		return domain.ErrTokenExpired
	}

	user, err := uc.users.GetByID(ctx, ev.UserID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	user.IsVerified = true
	user.UpdatedAt = time.Now()

	return uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.users.Update(ctx, user); err != nil {
			return fmt.Errorf("update user: %w", err)
		}
		if err := uc.tokens.DeleteEmailVerification(ctx, token); err != nil {
			return fmt.Errorf("delete email verification: %w", err)
		}
		_ = uc.cache.InvalidateProfile(ctx, user.ID)
		return nil
	})
}

func (uc *AuthUseCase) ResendVerification(ctx context.Context, rawEmail string) error {
	user, err := uc.users.GetByEmail(ctx, rawEmail)
	if err != nil {
		return nil // do not leak user existence
	}
	if user.IsVerified {
		return nil
	}

	token, err := generateSecureToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	ev := &domain.EmailVerification{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err = uc.tokens.SaveEmailVerification(ctx, ev); err != nil {
		return fmt.Errorf("save verification: %w", err)
	}
	_ = uc.publisher.Publish(ctx, domain.EventUserRegistered, map[string]string{
		"user_id":            user.ID.String(),
		"email":              string(user.Email),
		"username":           string(user.Username),
		"verification_token": token,
	})
	return nil
}

func (uc *AuthUseCase) ForgotPassword(ctx context.Context, rawEmail string) error {
	user, err := uc.users.GetByEmail(ctx, rawEmail)
	if err != nil {
		return nil // do not leak user existence
	}

	token, err := generateSecureToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	pr := &domain.PasswordReset{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err = uc.tokens.SavePasswordReset(ctx, pr); err != nil {
		return fmt.Errorf("save password reset: %w", err)
	}
	_ = uc.publisher.Publish(ctx, domain.EventPasswordResetRequested, map[string]string{
		"user_id": user.ID.String(),
		"email":   string(user.Email),
		"token":   token,
	})
	return nil
}

func (uc *AuthUseCase) ResetPassword(ctx context.Context, token, newRawPassword string) error {
	if _, err := domain.NewRawPassword(newRawPassword); err != nil {
		return err
	}

	pr, err := uc.tokens.GetPasswordReset(ctx, token)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrTokenInvalid
		}
		return fmt.Errorf("get password reset: %w", err)
	}
	if !pr.IsValid() {
		if pr.IsExpired() {
			return domain.ErrTokenExpired
		}
		return domain.ErrTokenInvalid
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newRawPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user, err := uc.users.GetByID(ctx, pr.UserID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now()

	return uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.users.Update(ctx, user); err != nil {
			return fmt.Errorf("update user: %w", err)
		}
		if err := uc.tokens.MarkPasswordResetUsed(ctx, token); err != nil {
			return fmt.Errorf("mark reset used: %w", err)
		}
		if err := uc.tokens.RevokeAllUserTokens(ctx, user.ID); err != nil {
			return fmt.Errorf("revoke tokens: %w", err)
		}
		_ = uc.cache.InvalidateProfile(ctx, user.ID)
		return nil
	})
}

func (uc *AuthUseCase) ChangePassword(ctx context.Context, userID uuid.UUID, oldRaw, newRaw string) error {
	if _, err := domain.NewRawPassword(newRaw); err != nil {
		return err
	}

	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldRaw)); err != nil {
		return domain.ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newRaw), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now()

	return uc.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := uc.users.Update(ctx, user); err != nil {
			return fmt.Errorf("update user: %w", err)
		}
		if err := uc.tokens.RevokeAllUserTokens(ctx, user.ID); err != nil {
			return fmt.Errorf("revoke tokens: %w", err)
		}
		_ = uc.cache.InvalidateProfile(ctx, user.ID)
		return nil
	})
}

func (uc *AuthUseCase) ValidateToken(ctx context.Context, accessToken string) (*Claims, error) {
	token, err := jwt.Parse(accessToken, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return uc.pubKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domain.ErrTokenExpired
		}
		return nil, domain.ErrTokenInvalid
	}

	mc, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrTokenInvalid
	}

	sub, _ := mc["sub"].(string)
	userID, err := uuid.Parse(sub)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}
	username, _ := mc["username"].(string)
	email, _ := mc["email"].(string)

	return &Claims{
		UserID:   userID,
		Username: username,
		Email:    email,
	}, nil
}

// ── helpers ────────────────────────────────────────────────────────────────

func (uc *AuthUseCase) issuePair(ctx context.Context, user *domain.User) (*TokenPair, error) {
	access, err := uc.issueAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}
	refresh, err := uc.issueRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}
	return &TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (uc *AuthUseCase) issueAccessToken(user *domain.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      user.ID.String(),
		"username": string(user.Username),
		"email":    string(user.Email),
		"iat":      now.Unix(),
		"exp":      now.Add(uc.accessTTL).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	return t.SignedString(uc.privKey)
}

func (uc *AuthUseCase) issueRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("rand read: %w", err)
	}
	rawToken := base64.URLEncoding.EncodeToString(raw)

	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(uc.refreshTTL),
		CreatedAt: time.Now(),
	}
	if err := uc.tokens.SaveRefreshToken(ctx, rt); err != nil {
		return "", err
	}
	return rawToken, nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
