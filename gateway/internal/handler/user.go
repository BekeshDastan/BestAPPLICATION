package handler

import (
	"net/http"

	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"github.com/bekesh/social/gateway/internal/middleware"
	"github.com/gin-gonic/gin"
)

type UserHandler struct{ uc userv1.UserServiceClient }

func NewUserHandler(uc userv1.UserServiceClient) *UserHandler { return &UserHandler{uc: uc} }

func (h *UserHandler) GetMe(c *gin.Context) {
	resp, err := h.uc.GetProfile(c.Request.Context(), &userv1.GetProfileRequest{
		UserId: middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	resp, err := h.uc.GetProfile(c.Request.Context(), &userv1.GetProfileRequest{
		UserId: c.Param("id"),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req struct {
		FullName  string `json:"full_name"`
		Bio       string `json:"bio"`
		IsPrivate bool   `json:"is_private"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.uc.UpdateProfile(c.Request.Context(), &userv1.UpdateProfileRequest{
		UserId:    middleware.CallerID(c),
		FullName:  req.FullName,
		Bio:       req.Bio,
		IsPrivate: req.IsPrivate,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *UserHandler) SearchUsers(c *gin.Context) {
	resp, err := h.uc.SearchUsers(c.Request.Context(), &userv1.SearchUsersRequest{
		Query:  c.Query("q"),
		Limit:  intQuery(c, "limit", 20),
		Offset: intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *UserHandler) Follow(c *gin.Context) {
	_, err := h.uc.Follow(c.Request.Context(), &userv1.FollowRequest{
		FollowerId: middleware.CallerID(c),
		FolloweeId: c.Param("id"),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *UserHandler) Unfollow(c *gin.Context) {
	_, err := h.uc.Unfollow(c.Request.Context(), &userv1.UnfollowRequest{
		FollowerId: middleware.CallerID(c),
		FolloweeId: c.Param("id"),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *UserHandler) ListFollowers(c *gin.Context) {
	resp, err := h.uc.ListFollowers(c.Request.Context(), &userv1.ListFollowersRequest{
		UserId: c.Param("id"),
		Limit:  intQuery(c, "limit", 20),
		Offset: intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *UserHandler) ListFollowing(c *gin.Context) {
	resp, err := h.uc.ListFollowing(c.Request.Context(), &userv1.ListFollowingRequest{
		UserId: c.Param("id"),
		Limit:  intQuery(c, "limit", 20),
		Offset: intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *UserHandler) IsFollowing(c *gin.Context) {
	resp, err := h.uc.IsFollowing(c.Request.Context(), &userv1.IsFollowingRequest{
		FollowerId: middleware.CallerID(c),
		FolloweeId: c.Param("id"),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
