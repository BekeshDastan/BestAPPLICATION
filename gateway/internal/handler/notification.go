package handler

import (
	"net/http"

	notifv1 "github.com/bekesh/social/gen/go/notification/v1"
	"github.com/bekesh/social/gateway/internal/middleware"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct{ uc notifv1.NotificationServiceClient }

func NewNotificationHandler(uc notifv1.NotificationServiceClient) *NotificationHandler {
	return &NotificationHandler{uc: uc}
}

func (h *NotificationHandler) List(c *gin.Context) {
	resp, err := h.uc.ListNotifications(c.Request.Context(), &notifv1.ListNotificationsRequest{
		UserId: middleware.CallerID(c),
		Limit:  intQuery(c, "limit", 20),
		Offset: intQuery(c, "offset", 0),
	})
	if err != nil {
		errResp(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"notifications": resp.Notifications,
		"total":         len(resp.Notifications),
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
