package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	chatv1  "github.com/bekesh/social/gen/go/chat/v1"
	notifv1 "github.com/bekesh/social/gen/go/notification/v1"
	postv1  "github.com/bekesh/social/gen/go/post/v1"
	storyv1 "github.com/bekesh/social/gen/go/story/v1"
	userv1  "github.com/bekesh/social/gen/go/user/v1"
	"github.com/bekesh/social/gateway/internal/middleware"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	user  userv1.UserServiceClient
	post  postv1.PostServiceClient
	chat  chatv1.ChatServiceClient
	story storyv1.StoryServiceClient
	notif notifv1.NotificationServiceClient
}

func NewAdminHandler(
	user userv1.UserServiceClient,
	post postv1.PostServiceClient,
	chat chatv1.ChatServiceClient,
	story storyv1.StoryServiceClient,
	notif notifv1.NotificationServiceClient,
) *AdminHandler {
	return &AdminHandler{user: user, post: post, chat: chat, story: story, notif: notif}
}

// GET /admin/me — return current user profile + is_admin flag.
// Reaching this handler already means the Admin middleware approved the caller.
func (h *AdminHandler) Me(c *gin.Context) {
	resp, err := h.user.GetProfile(c.Request.Context(), &userv1.GetProfileRequest{
		UserId: middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	u := resp.User
	c.JSON(http.StatusOK, gin.H{
		"id":          u.Id,
		"username":    u.Username,
		"email":       u.Email,
		"full_name":   u.FullName,
		"bio":         u.Bio,
		"avatar_url":  u.AvatarUrl,
		"is_verified": u.IsVerified,
		"is_admin":    middleware.IsAdmin(c),
	})
}

// GET /admin/stats — best-effort counts via SearchUsers/SearchPosts.
// Since proto has no Count RPC, we cap at 1000 each; deployments larger than
// that will under-report. Replace with dedicated Count RPCs in the future.
func (h *AdminHandler) Stats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	var totalUsers, totalPosts, activeStories int
	var userIDs []string

	if u, err := h.user.SearchUsers(ctx, &userv1.SearchUsersRequest{Limit: 1000}); err == nil {
		totalUsers = len(u.Users)
		userIDs = make([]string, 0, len(u.Users))
		for _, usr := range u.Users {
			userIDs = append(userIDs, usr.Id)
		}
	}
	if p, err := h.post.SearchPosts(ctx, &postv1.SearchPostsRequest{Limit: 1000}); err == nil {
		totalPosts = len(p.Posts)
	}
	// Active stories: story service has no global Count RPC, so we union
	// ListFollowingStories across every known user to enumerate live ones.
	if len(userIDs) > 0 {
		if s, err := h.story.ListFollowingStories(ctx, &storyv1.ListFollowingRequest{
			FollowingUserIds: userIDs,
		}); err == nil {
			activeStories = len(s.Stories)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_users":    totalUsers,
		"total_posts":    totalPosts,
		"active_stories": activeStories,
		"messages_today": 0, // no global count RPC yet
		"user_growth":    "+0%",
		"post_growth":    "+0%",
	})
}

// GET /admin/stats/registrations
func (h *AdminHandler) StatsRegistrations(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []gin.H{}})
}

// GET /admin/stats/posts
func (h *AdminHandler) StatsPosts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []gin.H{}})
}

// GET /admin/activity
func (h *AdminHandler) Activity(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"events": []gin.H{}})
}

// GET /admin/users/top
func (h *AdminHandler) TopUsers(c *gin.Context) {
	resp, err := h.user.SearchUsers(c.Request.Context(), &userv1.SearchUsersRequest{
		Query: "",
		Limit: 10,
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"users": []gin.H{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": resp.Users})
}

// GET /admin/users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page := intQuery(c, "page", 1)
	limit := intQuery(c, "limit", 20)
	if page < 1 {
		page = 1
	}
	resp, err := h.user.SearchUsers(c.Request.Context(), &userv1.SearchUsersRequest{
		Query:  c.DefaultQuery("q", ""),
		Limit:  limit,
		Offset: (page - 1) * limit,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"users": resp.Users,
		"total": int(limit) * 5, // approximate, no total count in proto
	})
}

// GET /admin/users/:id
func (h *AdminHandler) GetUser(c *gin.Context) {
	resp, err := h.user.GetProfile(c.Request.Context(), &userv1.GetProfileRequest{
		UserId: c.Param("id"),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.User)
}

// DELETE /admin/users/:id — bypasses password check via dedicated RPC.
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	_, err := h.user.AdminDeleteUser(c.Request.Context(), &userv1.AdminDeleteUserRequest{
		UserId: c.Param("id"),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// PUT /admin/users/:id/suspend — currently a no-op stub.
// Proper implementation requires an is_suspended column in users + a
// SuspendUser RPC. For now we just log so audit shows the intent.
func (h *AdminHandler) BanUser(c *gin.Context) {
	slog.Info("admin suspend (stub — no DB write)",
		"admin", middleware.CallerID(c),
		"target", c.Param("id"),
	)
	c.Status(http.StatusNoContent)
}

// GET /admin/posts
func (h *AdminHandler) ListPosts(c *gin.Context) {
	page := intQuery(c, "page", 1)
	limit := intQuery(c, "limit", 20)
	if page < 1 {
		page = 1
	}
	resp, err := h.post.SearchPosts(c.Request.Context(), &postv1.SearchPostsRequest{
		Query:  c.DefaultQuery("q", ""),
		Limit:  limit,
		Offset: (page - 1) * limit,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"posts": resp.Posts,
		"total": int(limit) * 5,
	})
}

// DELETE /admin/posts/:id — admin can delete ANY post.
// Post service enforces AuthorID match, so we look up the real author first
// and call DeletePost on its behalf.
func (h *AdminHandler) DeletePost(c *gin.Context) {
	postID := c.Param("id")
	get, err := h.post.GetPost(c.Request.Context(), &postv1.GetPostRequest{Id: postID})
	if err != nil {
		errResp(c, err)
		return
	}
	_, err = h.post.DeletePost(c.Request.Context(), &postv1.DeletePostRequest{
		Id:       postID,
		AuthorId: get.Post.AuthorId,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GET /admin/stories — best-effort: list active stories from top users.
// Story service has no "list all" RPC; we use ListFollowingStories with the
// full user list as a workaround.
func (h *AdminHandler) ListStories(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	usersResp, err := h.user.SearchUsers(ctx, &userv1.SearchUsersRequest{Limit: 500})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"stories": []gin.H{}, "total": 0})
		return
	}
	ids := make([]string, 0, len(usersResp.Users))
	for _, u := range usersResp.Users {
		ids = append(ids, u.Id)
	}
	if len(ids) == 0 {
		c.JSON(http.StatusOK, gin.H{"stories": []gin.H{}, "total": 0})
		return
	}
	resp, err := h.story.ListFollowingStories(ctx, &storyv1.ListFollowingRequest{
		FollowingUserIds: ids,
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"stories": []gin.H{}, "total": 0})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"stories": resp.Stories,
		"total":   len(resp.Stories),
	})
}

// DELETE /admin/stories/:id — admin can delete ANY story.
// Story service checks ownership, so we look up the real author first.
func (h *AdminHandler) DeleteStory(c *gin.Context) {
	storyID := c.Param("id")
	get, err := h.story.GetStory(c.Request.Context(), &storyv1.GetStoryRequest{
		StoryId: storyID,
		UserId:  middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	_, err = h.story.DeleteStory(c.Request.Context(), &storyv1.DeleteStoryRequest{
		StoryId: storyID,
		UserId:  get.Story.UserId,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GET /admin/reports
func (h *AdminHandler) ListReports(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"reports": []gin.H{}, "total": 0})
}

// PUT /admin/reports/:id/resolve
func (h *AdminHandler) ResolveReport(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// GET /admin/system/health — ping each gRPC service
func (h *AdminHandler) SystemHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	status := func(err error) string {
		if err == nil {
			return "healthy"
		}
		return "down"
	}

	// Use the zero UUID so per-service uuid.Parse() succeeds — we just want a
	// round-trip ping, not a real query. A non-UUID like "ping" returns
	// InvalidArgument and would be misreported as service-down.
	const pingUUID = "00000000-0000-0000-0000-000000000000"

	_, errUser := h.user.SearchUsers(ctx, &userv1.SearchUsersRequest{Limit: 1})
	_, errPost := h.post.SearchPosts(ctx, &postv1.SearchPostsRequest{Limit: 1})
	_, errChat := h.chat.ListConversations(ctx, &chatv1.ListConversationsRequest{UserId: pingUUID, Limit: 1})
	_, errStory := h.story.ListUserStories(ctx, &storyv1.ListUserStoriesRequest{UserId: pingUUID, Limit: 1})
	_, errNotif := h.notif.GetUnreadCount(ctx, &notifv1.GetUnreadCountRequest{UserId: pingUUID})

	c.JSON(http.StatusOK, gin.H{
		"services": gin.H{
			"user":         status(errUser),
			"post":         status(errPost),
			"chat":         status(errChat),
			"story":        status(errStory),
			"notification": status(errNotif),
			"gateway":      "healthy",
			"postgres":     "unknown",
			"redis":        "unknown",
			"nats":         "unknown",
			"minio":        "unknown",
			"mailhog":      "unknown",
		},
	})
}
