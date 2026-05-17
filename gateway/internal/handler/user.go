package handler

import (
	"context"
	"net/http"
	"time"

	postv1 "github.com/bekesh/social/gen/go/post/v1"
	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"github.com/bekesh/social/gateway/internal/middleware"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	uc   userv1.UserServiceClient
	post postv1.PostServiceClient
}

func NewUserHandler(uc userv1.UserServiceClient, post postv1.PostServiceClient) *UserHandler {
	return &UserHandler{uc: uc, post: post}
}

// profilePayload wraps a UserProfile with the social counters and the
// caller-relative follow/block flags that the frontend needs but the proto
// schema does not carry. Computed at the gateway via fan-out RPCs.
type profilePayload struct {
	*userv1.UserProfile
	PostsCount     int  `json:"posts_count"`
	FollowersCount int  `json:"followers_count"`
	FollowingCount int  `json:"following_count"`
	IsFollowing    bool `json:"is_following"`
}

// enrichProfile fans out to user-service (followers/following) and post-service
// (posts count) to fill in the counters. callerID may be empty for unauthed
// callers; in that case is_following is left false.
func (h *UserHandler) enrichProfile(ctx context.Context, u *userv1.UserProfile, callerID string) *profilePayload {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	out := &profilePayload{UserProfile: u}

	// Followers / following counts — pull up to 1000 each. Past that, the
	// number under-reports until we add a dedicated Count RPC.
	if r, err := h.uc.ListFollowers(ctx, &userv1.ListFollowersRequest{UserId: u.Id, Limit: 1000}); err == nil {
		out.FollowersCount = len(r.Users)
	}
	if r, err := h.uc.ListFollowing(ctx, &userv1.ListFollowingRequest{UserId: u.Id, Limit: 1000}); err == nil {
		out.FollowingCount = len(r.Users)
	}
	if r, err := h.post.ListUserPosts(ctx, &postv1.ListUserPostsRequest{AuthorId: u.Id, Limit: 1000}); err == nil {
		out.PostsCount = len(r.Posts)
	}

	if callerID != "" && callerID != u.Id {
		if r, err := h.uc.IsFollowing(ctx, &userv1.IsFollowingRequest{
			FollowerId: callerID,
			FolloweeId: u.Id,
		}); err == nil {
			out.IsFollowing = r.IsFollowing
		}
	}
	return out
}

func (h *UserHandler) GetMe(c *gin.Context) {
	callerID := middleware.CallerID(c)
	resp, err := h.uc.GetProfile(c.Request.Context(), &userv1.GetProfileRequest{
		UserId: callerID,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, h.enrichProfile(c.Request.Context(), resp.User, callerID))
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	resp, err := h.uc.GetProfile(c.Request.Context(), &userv1.GetProfileRequest{
		UserId: c.Param("id"),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, h.enrichProfile(c.Request.Context(), resp.User, middleware.CallerID(c)))
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
	callerID := middleware.CallerID(c)
	resp, err := h.uc.UpdateProfile(c.Request.Context(), &userv1.UpdateProfileRequest{
		UserId:    callerID,
		FullName:  req.FullName,
		Bio:       req.Bio,
		IsPrivate: req.IsPrivate,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, h.enrichProfile(c.Request.Context(), resp.User, callerID))
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
	c.JSON(http.StatusOK, gin.H{"users": h.attachFollowState(c.Request.Context(), middleware.CallerID(c), resp.Users)})
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

	filtered := make([]*userv1.UserProfile, 0, len(resp.Users))
	for _, u := range resp.Users {
		if u.Id == callerID {
			continue
		}
		filtered = append(filtered, u)
	}
	c.JSON(http.StatusOK, gin.H{"users": h.attachFollowState(c.Request.Context(), callerID, filtered)})
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
	c.JSON(http.StatusOK, gin.H{"users": h.attachFollowState(c.Request.Context(), middleware.CallerID(c), resp.Users)})
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
	c.JSON(http.StatusOK, gin.H{"users": h.attachFollowState(c.Request.Context(), middleware.CallerID(c), resp.Users)})
}

// attachFollowState marks each user with is_following relative to the caller.
// Batched: one ListFollowing call for the caller, then in-memory set lookup.
// Pure UserProfile would otherwise force the frontend to issue N is-following
// requests per modal render.
func (h *UserHandler) attachFollowState(ctx context.Context, callerID string, users []*userv1.UserProfile) []any {
	out := make([]any, 0, len(users))
	if callerID == "" {
		for _, u := range users {
			out = append(out, u)
		}
		return out
	}

	following := map[string]struct{}{}
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if r, err := h.uc.ListFollowing(ctx2, &userv1.ListFollowingRequest{
		UserId: callerID,
		Limit:  1000,
	}); err == nil {
		for _, u := range r.Users {
			following[u.Id] = struct{}{}
		}
	}

	for _, u := range users {
		_, isFollowing := following[u.Id]
		out = append(out, struct {
			*userv1.UserProfile
			IsFollowing bool `json:"is_following"`
		}{u, isFollowing})
	}
	return out
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
	callerID := middleware.CallerID(c)
	resp, err := h.uc.UpdateAvatar(c.Request.Context(), &userv1.UpdateAvatarRequest{
		UserId:    callerID,
		AvatarUrl: req.AvatarURL,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, h.enrichProfile(c.Request.Context(), resp.User, callerID))
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
