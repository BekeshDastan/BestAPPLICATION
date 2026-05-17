package handler

import (
	"net/http"

	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"github.com/bekesh/social/gateway/internal/middleware"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct{ uc userv1.UserServiceClient }

func NewAuthHandler(uc userv1.UserServiceClient) *AuthHandler { return &AuthHandler{uc: uc} }

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Username string `json:"username"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.uc.Register(c.Request.Context(), &userv1.RegisterRequest{
		Email: req.Email, Username: req.Username,
		Password: req.Password, FullName: req.FullName,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"user":          resp.User,
		"access_token":  resp.Tokens.GetAccessToken(),
		"refresh_token": resp.Tokens.GetRefreshToken(),
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.uc.Login(c.Request.Context(), &userv1.LoginRequest{
		Email: req.Email, Password: req.Password,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user":          resp.User,
		"access_token":  resp.Tokens.GetAccessToken(),
		"refresh_token": resp.Tokens.GetRefreshToken(),
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.uc.Logout(c.Request.Context(), &userv1.LogoutRequest{RefreshToken: req.RefreshToken})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.uc.RefreshToken(c.Request.Context(), &userv1.RefreshTokenRequest{RefreshToken: req.RefreshToken})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  resp.Tokens.GetAccessToken(),
		"refresh_token": resp.Tokens.GetRefreshToken(),
	})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.uc.ForgotPassword(c.Request.Context(), &userv1.ForgotPasswordRequest{Email: req.Email})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "reset email sent"})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.uc.ResetPassword(c.Request.Context(), &userv1.ResetPasswordRequest{
		Token: req.Token, NewPassword: req.NewPassword,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password reset"})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.uc.VerifyEmail(c.Request.Context(), &userv1.VerifyEmailRequest{Token: req.Token})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "email verified"})
}

func (h *AuthHandler) ResendVerification(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	c.ShouldBindJSON(&req)
	_, err := h.uc.ResendVerification(c.Request.Context(), &userv1.ResendVerificationRequest{Email: req.Email})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "verification email sent"})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.uc.ChangePassword(c.Request.Context(), &userv1.ChangePasswordRequest{
		UserId:      middleware.CallerID(c),
		OldPassword: req.CurrentPassword,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
