package handler

import (
	"context"
	"net/http"
	"strings"

	notifv1 "github.com/bekesh/social/gen/go/notification/v1"
	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"github.com/bekesh/social/gateway/internal/middleware"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	uc   notifv1.NotificationServiceClient
	user userv1.UserServiceClient
}

func NewNotificationHandler(uc notifv1.NotificationServiceClient, user userv1.UserServiceClient) *NotificationHandler {
	return &NotificationHandler{uc: uc, user: user}
}

type notifActorInfo struct {
	ID        string `json:"id"`
	Username  string `json:"username,omitempty"`
	FullName  string `json:"full_name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type enrichedNotif struct {
	ID            string          `json:"id"`
	UserID        string          `json:"user_id"`
	ActorID       string          `json:"actor_id"`
	Type          string          `json:"type"`
	ReferenceID   string          `json:"reference_id,omitempty"`
	ReferenceType string          `json:"reference_type,omitempty"`
	Message       string          `json:"message,omitempty"`
	IsRead        bool            `json:"is_read"`
	CreatedAt     int64           `json:"created_at"`
	Actor         *notifActorInfo `json:"actor,omitempty"`
}

func (h *NotificationHandler) fetchActorMap(ctx context.Context, ids []string) map[string]*notifActorInfo {
	seen := make(map[string]bool)
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != "" && !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	type result struct {
		id   string
		info *notifActorInfo
	}
	ch := make(chan result, len(unique))
	for _, id := range unique {
		id := id
		go func() {
			r, err := h.user.GetProfile(ctx, &userv1.GetProfileRequest{UserId: id})
			if err != nil || r.User == nil {
				ch <- result{id, nil}
				return
			}
			ch <- result{id, &notifActorInfo{
				ID:        r.User.Id,
				Username:  r.User.Username,
				FullName:  r.User.FullName,
				AvatarURL: r.User.AvatarUrl,
			}}
		}()
	}
	out := make(map[string]*notifActorInfo, len(unique))
	for range unique {
		r := <-ch
		if r.info != nil {
			out[r.id] = r.info
		}
	}
	return out
}

func (h *NotificationHandler) List(c *gin.Context) {
	typeFilter := strings.TrimSpace(c.Query("type"))
	limit := intQuery(c, "limit", 20)

	// Fetch more when a type filter is active so we have enough to filter from.
	fetchLimit := limit
	if typeFilter != "" {
		fetchLimit = 100
	}

	resp, err := h.uc.ListNotifications(c.Request.Context(), &notifv1.ListNotificationsRequest{
		UserId: middleware.CallerID(c),
		Limit:  fetchLimit,
		Offset: intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}

	notifications := resp.Notifications
	if typeFilter != "" {
		filtered := make([]*notifv1.NotificationProto, 0, int(limit))
		for _, n := range notifications {
			if strings.Contains(n.Type, typeFilter) {
				filtered = append(filtered, n)
				if int32(len(filtered)) >= limit {
					break
				}
			}
		}
		notifications = filtered
	}

	// Enrich with actor profile info.
	actorIDs := make([]string, len(notifications))
	for i, n := range notifications {
		actorIDs[i] = n.ActorId
	}
	actors := h.fetchActorMap(c.Request.Context(), actorIDs)

	out := make([]enrichedNotif, len(notifications))
	for i, n := range notifications {
		out[i] = enrichedNotif{
			ID:            n.Id,
			UserID:        n.UserId,
			ActorID:       n.ActorId,
			Type:          n.Type,
			ReferenceID:   n.ReferenceId,
			ReferenceType: n.ReferenceType,
			Message:       n.Message,
			IsRead:        n.IsRead,
			CreatedAt:     n.CreatedAt,
			Actor:         actors[n.ActorId],
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": out,
		"total":         len(out),
	})
}

func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	resp, err := h.uc.GetUnreadCount(c.Request.Context(), &notifv1.GetUnreadCountRequest{
		UserId: middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": resp.Count})
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	_, err := h.uc.MarkAsRead(c.Request.Context(), &notifv1.MarkAsReadRequest{
		Id:     c.Param("id"),
		UserId: middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	_, err := h.uc.MarkAllAsRead(c.Request.Context(), &notifv1.MarkAllAsReadRequest{
		UserId: middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *NotificationHandler) Delete(c *gin.Context) {
	_, err := h.uc.DeleteNotification(c.Request.Context(), &notifv1.DeleteNotificationRequest{
		Id:     c.Param("id"),
		UserId: middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *NotificationHandler) GetSettings(c *gin.Context) {
	resp, err := h.uc.GetPreferences(c.Request.Context(), &notifv1.GetPreferencesRequest{
		UserId: middleware.CallerID(c),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	dict := make(map[string]map[string]bool, len(resp.Preferences))
	for _, p := range resp.Preferences {
		dict[p.Type] = map[string]bool{
			"push":  p.PushEnabled,
			"email": p.EmailEnabled,
		}
	}
	c.JSON(http.StatusOK, dict)
}

func (h *NotificationHandler) UpdateSetting(c *gin.Context) {
	var req map[string]struct {
		Push  bool `json:"push"`
		Email bool `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	callerID := middleware.CallerID(c)
	for typ, s := range req {
		_, err := h.uc.UpdatePreference(c.Request.Context(), &notifv1.UpdatePreferenceRequest{
			UserId:       callerID,
			Type:         typ,
			EmailEnabled: s.Email,
			PushEnabled:  s.Push,
		})
		if err != nil {
			errResp(c, err)
			return
		}
	}
	c.Status(http.StatusNoContent)
}
