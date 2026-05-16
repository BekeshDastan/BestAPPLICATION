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
	c.JSON(http.StatusOK, resp.User)
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	resp, err := h.uc.GetProfile(c.Request.Context(), &userv1.GetProfileRequest{
		UserId: c.Param("id"),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.User)
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
	c.JSON(http.StatusOK, resp.User)
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

// SuggestUsers returns a simple list of users the caller is not yet following.
// Backed by SearchUsers with empty query (no dedicated RPC yet).
func (h *UserHandler) SuggestUsers(c *gin.Context) {
	callerID := middleware.CallerID(c)

	resp, err := h.uc.SearchUsers(c.Request.Context(), &userv1.SearchUsersRequest{
		Query:  "",
		Limit:  intQuery(c, "limit", 10),
		Offset: 0,
	})
	if err != nil {
		errResp(c, err)
		return
	}

	out := make([]any, 0, len(resp.Users))
	for _, u := range resp.Users {
		if u.Id == callerID {
			continue
		}
		out = append(out, u)
	}
	c.JSON(http.StatusOK, gin.H{"users": out})
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

func (h *UserHandler) UpdateAvatar(c *gin.Context) {
	var req struct {
		AvatarURL string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.uc.UpdateAvatar(c.Request.Context(), &userv1.UpdateAvatarRequest{
		UserId:    middleware.CallerID(c),
		AvatarUrl: req.AvatarURL,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.User)
}

func (h *UserHandler) DeleteAccount(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	c.ShouldBindJSON(&req)
	_, err := h.uc.DeleteAccount(c.Request.Context(), &userv1.DeleteAccountRequest{
		UserId:   middleware.CallerID(c),
		Password: req.Password,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *UserHandler) Block(c *gin.Context) {
	_, err := h.uc.BlockUser(c.Request.Context(), &userv1.BlockUserRequest{
		BlockerId: middleware.CallerID(c),
		BlockedId: c.Param("id"),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *UserHandler) Unblock(c *gin.Context) {
	_, err := h.uc.UnblockUser(c.Request.Context(), &userv1.UnblockUserRequest{
		BlockerId: middleware.CallerID(c),
		BlockedId: c.Param("id"),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
