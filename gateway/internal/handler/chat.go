package handler

import (
	"net/http"

	chatv1 "github.com/bekesh/social/gen/go/chat/v1"
	"github.com/bekesh/social/gateway/internal/middleware"
	"github.com/gin-gonic/gin"
)

type ChatHandler struct{ uc chatv1.ChatServiceClient }

func NewChatHandler(uc chatv1.ChatServiceClient) *ChatHandler { return &ChatHandler{uc: uc} }

func (h *ChatHandler) CreateConversation(c *gin.Context) {
	var req struct {
		MemberIDs []string `json:"member_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.uc.CreateConversation(c.Request.Context(), &chatv1.CreateConversationRequest{
		CreatorId: middleware.CallerID(c),
		MemberIds: req.MemberIDs,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *ChatHandler) ListConversations(c *gin.Context) {
	resp, err := h.uc.ListConversations(c.Request.Context(), &chatv1.ListConversationsRequest{
		UserId: middleware.CallerID(c),
		Limit:  intQuery(c, "limit", 20),
		Offset: intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ChatHandler) GetConversation(c *gin.Context) {
	resp, err := h.uc.GetConversation(c.Request.Context(), &chatv1.GetConversationRequest{
		ConversationId: c.Param("id"),
		UserId:         middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	_, err := h.uc.DeleteConversation(c.Request.Context(), &chatv1.DeleteConversationRequest{
		ConversationId: c.Param("id"),
		UserId:         middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	var req struct {
		Text      string `json:"text"`
		MediaURL  string `json:"media_url"`
		ReplyToID string `json:"reply_to_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.uc.SendMessage(c.Request.Context(), &chatv1.SendMessageRequest{
		ConversationId: c.Param("id"),
		SenderId:       middleware.CallerID(c),
		Text:           req.Text,
		MediaUrl:       req.MediaURL,
		ReplyToId:      req.ReplyToID,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *ChatHandler) ListMessages(c *gin.Context) {
	resp, err := h.uc.ListMessages(c.Request.Context(), &chatv1.ListMessagesRequest{
		ConversationId: c.Param("id"),
		UserId:         middleware.CallerID(c),
		Limit:          intQuery(c, "limit", 50),
		Offset:         intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ChatHandler) DeleteMessage(c *gin.Context) {
	_, err := h.uc.DeleteMessage(c.Request.Context(), &chatv1.DeleteMessageRequest{
		MessageId: c.Param("msg_id"),
		SenderId:  middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
