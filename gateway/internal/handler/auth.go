package handler

import (
	"net/http"

	userv1 "github.com/bekesh/social/gen/go/user/v1"
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
	c.JSON(http.StatusCreated, resp)
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
	c.JSON(http.StatusOK, resp)
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
	c.JSON(http.StatusOK, resp)
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
