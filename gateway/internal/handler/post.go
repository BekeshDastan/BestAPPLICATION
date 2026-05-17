package handler

import (
	"net/http"
	"strings"

	postv1 "github.com/bekesh/social/gen/go/post/v1"
	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"github.com/bekesh/social/gateway/internal/middleware"
	"github.com/gin-gonic/gin"
)

type PostHandler struct {
	uc   postv1.PostServiceClient
	user userv1.UserServiceClient
}

func NewPostHandler(uc postv1.PostServiceClient, user userv1.UserServiceClient) *PostHandler {
	return &PostHandler{uc: uc, user: user}
}

func (h *PostHandler) Create(c *gin.Context) {
	var req struct {
		Caption   string   `json:"caption"`
		MediaURLs []string `json:"media_urls"`
		Tags      []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.uc.CreatePost(c.Request.Context(), &postv1.CreatePostRequest{
		AuthorId:  middleware.CallerID(c),
		Caption:   req.Caption,
		MediaUrls: req.MediaURLs,
		Tags:      req.Tags,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *PostHandler) Get(c *gin.Context) {
	resp, err := h.uc.GetPost(c.Request.Context(), &postv1.GetPostRequest{Id: c.Param("id")})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) Update(c *gin.Context) {
	var req struct {
		Caption string   `json:"caption"`
		Tags    []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.uc.UpdatePost(c.Request.Context(), &postv1.UpdatePostRequest{
		Id:       c.Param("id"),
		AuthorId: middleware.CallerID(c),
		Caption:  req.Caption,
		Tags:     req.Tags,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) Delete(c *gin.Context) {
	_, err := h.uc.DeletePost(c.Request.Context(), &postv1.DeletePostRequest{
		Id:       c.Param("id"),
		AuthorId: middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PostHandler) Feed(c *gin.Context) {
	// If the client supplies following_ids explicitly, use them. Otherwise
	// fall back to fetching the caller's following list from user-service so
	// the frontend doesn't need to do it manually.
	var followingIDs []string
	if raw := strings.TrimSpace(c.Query("following_ids")); raw != "" {
		followingIDs = strings.Split(raw, ",")
	} else {
		callerID := middleware.CallerID(c)
		fr, err := h.user.ListFollowing(c.Request.Context(), &userv1.ListFollowingRequest{
			UserId: callerID,
			Limit:  500,
		})
		if err == nil {
			followingIDs = make([]string, 0, len(fr.Users)+1)
			followingIDs = append(followingIDs, callerID) // include own posts
			for _, u := range fr.Users {
				followingIDs = append(followingIDs, u.Id)
			}
		}
	}
	resp, err := h.uc.GetFeed(c.Request.Context(), &postv1.GetFeedRequest{
		FollowingIds: followingIDs,
		Limit:        intQuery(c, "limit", 20),
		Offset:       intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) ListUserPosts(c *gin.Context) {
	resp, err := h.uc.ListUserPosts(c.Request.Context(), &postv1.ListUserPostsRequest{
		AuthorId: c.Param("id"),
		Limit:    intQuery(c, "limit", 20),
		Offset:   intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) Search(c *gin.Context) {
	resp, err := h.uc.SearchPosts(c.Request.Context(), &postv1.SearchPostsRequest{
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

func (h *PostHandler) Like(c *gin.Context) {
	_, err := h.uc.LikePost(c.Request.Context(), &postv1.LikePostRequest{
		PostId: c.Param("id"),
		UserId: middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PostHandler) Unlike(c *gin.Context) {
	_, err := h.uc.UnlikePost(c.Request.Context(), &postv1.UnlikePostRequest{
		PostId: c.Param("id"),
		UserId: middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PostHandler) AddComment(c *gin.Context) {
	var req struct {
		Body string `json:"body"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.uc.AddComment(c.Request.Context(), &postv1.AddCommentRequest{
		PostId:   c.Param("id"),
		AuthorId: middleware.CallerID(c),
		Body:     req.Body,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *PostHandler) ListComments(c *gin.Context) {
	resp, err := h.uc.ListComments(c.Request.Context(), &postv1.ListCommentsRequest{
		PostId: c.Param("id"),
		Limit:  intQuery(c, "limit", 20),
		Offset: intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) DeleteComment(c *gin.Context) {
	_, err := h.uc.DeleteComment(c.Request.Context(), &postv1.DeleteCommentRequest{
		CommentId:   c.Param("comment_id"),
		RequesterId: middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PostHandler) SavePost(c *gin.Context) {
	_, err := h.uc.SavePost(c.Request.Context(), &postv1.SavePostRequest{
		PostId: c.Param("id"),
		UserId: middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PostHandler) UnsavePost(c *gin.Context) {
	_, err := h.uc.UnsavePost(c.Request.Context(), &postv1.UnsavePostRequest{
		PostId: c.Param("id"),
		UserId: middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PostHandler) GetSavedPosts(c *gin.Context) {
	resp, err := h.uc.GetSavedPosts(c.Request.Context(), &postv1.GetSavedPostsRequest{
		UserId: middleware.CallerID(c),
		Limit:  intQuery(c, "limit", 20),
		Offset: intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"posts": resp.Posts, "next_cursor": nil})
}

func (h *PostHandler) ReportPost(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
