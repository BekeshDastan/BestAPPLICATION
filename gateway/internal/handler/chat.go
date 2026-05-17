package handler

import (
	"context"
	"net/http"
	"sync"
	"time"

	chatv1 "github.com/bekesh/social/gen/go/chat/v1"
	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"github.com/bekesh/social/gateway/internal/middleware"
	"github.com/gin-gonic/gin"
)

// profileCache is a tiny TTL cache for user profiles fetched during
// conversation enrichment. Kills the N+1 RPC pattern in ListConversations
// when the same user appears across many direct conversations.
type profileCache struct {
	mu   sync.RWMutex
	data map[string]profileEntry
	ttl  time.Duration
}

type profileEntry struct {
	user      *userv1.UserProfile
	expiresAt time.Time
}

func newProfileCache(ttl time.Duration) *profileCache {
	return &profileCache{data: make(map[string]profileEntry), ttl: ttl}
}

func (c *profileCache) get(uid string) (*userv1.UserProfile, bool) {
	c.mu.RLock()
	e, ok := c.data[uid]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.user, true
}

func (c *profileCache) set(uid string, u *userv1.UserProfile) {
	c.mu.Lock()
	c.data[uid] = profileEntry{user: u, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

type ChatHandler struct {
	uc       chatv1.ChatServiceClient
	user     userv1.UserServiceClient
	hub      *Hub
	profiles *profileCache
}

func NewChatHandler(uc chatv1.ChatServiceClient, user userv1.UserServiceClient, hub *Hub) *ChatHandler {
	return &ChatHandler{
		uc:       uc,
		user:     user,
		hub:      hub,
		profiles: newProfileCache(60 * time.Second),
	}
}

// fetchProfile returns a profile from cache or fetches it via gRPC.
func (h *ChatHandler) fetchProfile(ctx context.Context, uid string) *userv1.UserProfile {
	if u, ok := h.profiles.get(uid); ok {
		return u
	}
	resp, err := h.user.GetProfile(ctx, &userv1.GetProfileRequest{UserId: uid})
	if err != nil || resp.User == nil {
		return nil
	}
	h.profiles.set(uid, resp.User)
	return resp.User
}

// enrichConv adds is_group and other_user fields for direct conversations.
func (h *ChatHandler) enrichConv(ctx context.Context, conv *chatv1.ConversationProto, callerID string) map[string]any {
	out := map[string]any{
		"id":              conv.Id,
		"type":            conv.Type,
		"name":            conv.Name,
		"avatar_url":      conv.AvatarUrl,
		"created_by":      conv.CreatedBy,
		"last_message_at": conv.LastMessageAt,
		"created_at":      conv.CreatedAt,
		"member_ids":      conv.MemberIds,
		"is_group":        conv.Type == "group",
	}
	if conv.Type != "group" {
		for _, uid := range conv.MemberIds {
			if uid != callerID {
				if u := h.fetchProfile(ctx, uid); u != nil {
					out["other_user"] = u
				}
				break
			}
		}
	}
	return out
}

// ── Conversations ──────────────────────────────────────────────────────────

func (h *ChatHandler) CreateConversation(c *gin.Context) {
	var req struct {
		MemberIDs []string `json:"member_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	callerID := middleware.CallerID(c)
	resp, err := h.uc.CreateConversation(c.Request.Context(), &chatv1.CreateConversationRequest{
		CreatorId: callerID,
		MemberIds: req.MemberIDs,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	enriched := h.enrichConv(c.Request.Context(), resp.Conversation, callerID)
	c.JSON(http.StatusCreated, gin.H{"conversation": enriched})
}

func (h *ChatHandler) ListConversations(c *gin.Context) {
	callerID := middleware.CallerID(c)
	resp, err := h.uc.ListConversations(c.Request.Context(), &chatv1.ListConversationsRequest{
		UserId: callerID,
		Limit:  intQuery(c, "limit", 20),
		Offset: intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	enriched := make([]map[string]any, len(resp.Conversations))
	for i, conv := range resp.Conversations {
		enriched[i] = h.enrichConv(c.Request.Context(), conv, callerID)
	}
	c.JSON(http.StatusOK, gin.H{"conversations": enriched})
}

func (h *ChatHandler) GetConversation(c *gin.Context) {
	callerID := middleware.CallerID(c)
	resp, err := h.uc.GetConversation(c.Request.Context(), &chatv1.GetConversationRequest{
		ConversationId: c.Param("id"),
		UserId:         callerID,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, h.enrichConv(c.Request.Context(), resp.Conversation, callerID))
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

func (h *ChatHandler) MarkConversationRead(c *gin.Context) {
	convID := c.Param("id")
	_, err := h.uc.MarkConversationRead(c.Request.Context(), &chatv1.MarkConvReadRequest{
		ConversationId: convID,
		UserId:         middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	h.hub.Broadcast(convID, WsEvent{Type: "read", ConversationID: convID})
	c.Status(http.StatusNoContent)
}

// ── Group management ───────────────────────────────────────────────────────

func (h *ChatHandler) UpdateGroup(c *gin.Context) {
	var req struct {
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.uc.UpdateGroupInfo(c.Request.Context(), &chatv1.UpdateGroupInfoRequest{
		ConversationId: c.Param("id"),
		RequesterId:    middleware.CallerID(c),
		Name:           req.Name,
		AvatarUrl:      req.AvatarURL,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ChatHandler) RemoveParticipant(c *gin.Context) {
	_, err := h.uc.RemoveParticipant(c.Request.Context(), &chatv1.RemoveParticipantRequest{
		ConversationId: c.Param("id"),
		RequesterId:    middleware.CallerID(c),
		UserId:         c.Param("uid"),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ChatHandler) LeaveGroup(c *gin.Context) {
	_, err := h.uc.LeaveGroup(c.Request.Context(), &chatv1.LeaveGroupRequest{
		ConversationId: c.Param("id"),
		UserId:         middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ChatHandler) AddParticipant(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.uc.AddParticipant(c.Request.Context(), &chatv1.AddParticipantRequest{
		ConversationId: c.Param("id"),
		RequesterId:    middleware.CallerID(c),
		UserId:         req.UserID,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ── Messages (nested under /chats/:id/messages) ────────────────────────────

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
	convID := c.Param("id")
	resp, err := h.uc.SendMessage(c.Request.Context(), &chatv1.SendMessageRequest{
		ConversationId: convID,
		SenderId:       middleware.CallerID(c),
		Text:           req.Text,
		MediaUrl:       req.MediaURL,
		ReplyToId:      req.ReplyToID,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	h.hub.Broadcast(convID, WsEvent{
		Type:           "message",
		ConversationID: convID,
		Payload:        resp.Message,
	})
	c.JSON(http.StatusCreated, resp.Message)
}

func (h *ChatHandler) ListMessages(c *gin.Context) {
	resp, err := h.uc.ListMessages(c.Request.Context(), &chatv1.ListMessagesRequest{
		ConversationId: c.Param("id"),
		UserId:         middleware.CallerID(c),
		Limit:          intQuery(c, "limit", 30),
		Offset:         intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": resp.Messages})
}

func (h *ChatHandler) DeleteMessageNested(c *gin.Context) {
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

// ── Flat message routes (/messages/*) ─────────────────────────────────────

// POST /messages  { conversation_id, text, reply_to_id, media_url }
func (h *ChatHandler) FlatSendMessage(c *gin.Context) {
	var req struct {
		ConversationID string `json:"conversation_id"`
		Text           string `json:"text"`
		MediaURL       string `json:"media_url"`
		ReplyToID      string `json:"reply_to_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.uc.SendMessage(c.Request.Context(), &chatv1.SendMessageRequest{
		ConversationId: req.ConversationID,
		SenderId:       middleware.CallerID(c),
		Text:           req.Text,
		MediaUrl:       req.MediaURL,
		ReplyToId:      req.ReplyToID,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	h.hub.Broadcast(req.ConversationID, WsEvent{
		Type:           "message",
		ConversationID: req.ConversationID,
		Payload:        resp.Message,
	})
	c.JSON(http.StatusCreated, resp.Message)
}

// PUT /messages/:id  { text }
func (h *ChatHandler) EditMessage(c *gin.Context) {
	var req struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.uc.EditMessage(c.Request.Context(), &chatv1.EditMessageRequest{
		MessageId: c.Param("id"),
		SenderId:  middleware.CallerID(c),
		Text:      req.Text,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	if resp.Message != nil {
		h.hub.Broadcast(resp.Message.ConversationId, WsEvent{
			Type:           "message_edited",
			ConversationID: resp.Message.ConversationId,
			MessageID:      resp.Message.Id,
			Text:           resp.Message.Text,
		})
	}
	c.JSON(http.StatusOK, resp.Message)
}

// DELETE /messages/:id
func (h *ChatHandler) DeleteMessage(c *gin.Context) {
	_, err := h.uc.DeleteMessage(c.Request.Context(), &chatv1.DeleteMessageRequest{
		MessageId: c.Param("id"),
		SenderId:  middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// POST /messages/:id/pin
func (h *ChatHandler) PinMessage(c *gin.Context) {
	var req struct {
		Pinned bool `json:"pinned"`
	}
	c.ShouldBindJSON(&req)
	_, err := h.uc.PinMessage(c.Request.Context(), &chatv1.PinMessageRequest{
		MessageId: c.Param("id"),
		UserId:    middleware.CallerID(c),
		Pinned:    req.Pinned,
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
