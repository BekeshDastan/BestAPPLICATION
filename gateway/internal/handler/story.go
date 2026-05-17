package handler

import (
	"net/http"

	storyv1 "github.com/bekesh/social/gen/go/story/v1"
	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"github.com/bekesh/social/gateway/internal/middleware"
	"github.com/gin-gonic/gin"
)

type StoryHandler struct {
	sc storyv1.StoryServiceClient
	uc userv1.UserServiceClient
}

func NewStoryHandler(sc storyv1.StoryServiceClient, uc userv1.UserServiceClient) *StoryHandler {
	return &StoryHandler{sc: sc, uc: uc}
}

func (h *StoryHandler) Create(c *gin.Context) {
	var req struct {
		MediaURL  string `json:"media_url"`
		MediaType string `json:"media_type"`
		Caption   string `json:"caption"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.sc.CreateStory(c.Request.Context(), &storyv1.CreateStoryRequest{
		UserId:    middleware.CallerID(c),
		MediaUrl:  req.MediaURL,
		MediaType: req.MediaType,
		Caption:   req.Caption,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *StoryHandler) Get(c *gin.Context) {
	resp, err := h.sc.GetStory(c.Request.Context(), &storyv1.GetStoryRequest{
		StoryId: c.Param("id"),
		UserId:  middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *StoryHandler) Delete(c *gin.Context) {
	_, err := h.sc.DeleteStory(c.Request.Context(), &storyv1.DeleteStoryRequest{
		StoryId: c.Param("id"),
		UserId:  middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *StoryHandler) ListUser(c *gin.Context) {
	resp, err := h.sc.ListUserStories(c.Request.Context(), &storyv1.ListUserStoriesRequest{
		UserId: c.Param("user_id"),
		Limit:  intQuery(c, "limit", 20),
		Offset: intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *StoryHandler) ListFollowing(c *gin.Context) {
	callerID := middleware.CallerID(c)
	followingResp, err := h.uc.ListFollowing(c.Request.Context(), &userv1.ListFollowingRequest{
		UserId: callerID,
		Limit:  50,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	ids := []string{callerID}
	for _, u := range followingResp.GetUsers() {
		ids = append(ids, u.GetId())
	}
	resp, err := h.sc.ListFollowingStories(c.Request.Context(), &storyv1.ListFollowingRequest{
		FollowingUserIds: ids,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *StoryHandler) MarkViewed(c *gin.Context) {
	_, err := h.sc.MarkStoryViewed(c.Request.Context(), &storyv1.MarkViewedRequest{
		StoryId:  c.Param("id"),
		ViewerId: middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *StoryHandler) ListViewers(c *gin.Context) {
	resp, err := h.sc.ListStoryViewers(c.Request.Context(), &storyv1.ListViewersRequest{
		StoryId: c.Param("id"),
		UserId:  middleware.CallerID(c),
		Limit:   intQuery(c, "limit", 30),
		Offset:  intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *StoryHandler) Reply(c *gin.Context) {
	var req struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.sc.ReplyToStory(c.Request.Context(), &storyv1.ReplyToStoryRequest{
		StoryId: c.Param("id"),
		UserId:  middleware.CallerID(c),
		Text:    req.Text,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *StoryHandler) AddReaction(c *gin.Context) {
	var req struct {
		Emoji string `json:"emoji"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.sc.AddReaction(c.Request.Context(), &storyv1.AddReactionRequest{
		StoryId: c.Param("id"),
		UserId:  middleware.CallerID(c),
		Emoji:   req.Emoji,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *StoryHandler) RemoveReaction(c *gin.Context) {
	_, err := h.sc.RemoveReaction(c.Request.Context(), &storyv1.RemoveReactionRequest{
		StoryId: c.Param("id"),
		UserId:  middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *StoryHandler) Analytics(c *gin.Context) {
	resp, err := h.sc.GetStoryAnalytics(c.Request.Context(), &storyv1.GetStoryAnalyticsRequest{
		StoryId: c.Param("id"),
		UserId:  middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *StoryHandler) CreateHighlight(c *gin.Context) {
	var req struct {
		Title    string `json:"title"`
		CoverURL string `json:"cover_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.sc.CreateHighlight(c.Request.Context(), &storyv1.CreateHighlightRequest{
		UserId:   middleware.CallerID(c),
		Title:    req.Title,
		CoverUrl: req.CoverURL,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *StoryHandler) AddToHighlight(c *gin.Context) {
	var req struct {
		StoryID string `json:"story_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.sc.AddToHighlight(c.Request.Context(), &storyv1.AddToHighlightRequest{
		HighlightId: c.Param("id"),
		StoryId:     req.StoryID,
		UserId:      middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *StoryHandler) RemoveFromHighlight(c *gin.Context) {
	_, err := h.sc.RemoveFromHighlight(c.Request.Context(), &storyv1.RemoveFromHighlightRequest{
		HighlightId: c.Param("id"),
		StoryId:     c.Param("story_id"),
		UserId:      middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *StoryHandler) ListHighlights(c *gin.Context) {
	resp, err := h.sc.ListHighlights(c.Request.Context(), &storyv1.ListHighlightsRequest{
		UserId: c.Param("user_id"),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *StoryHandler) DeleteHighlight(c *gin.Context) {
	_, err := h.sc.DeleteHighlight(c.Request.Context(), &storyv1.DeleteHighlightRequest{
		HighlightId: c.Param("id"),
		UserId:      middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
