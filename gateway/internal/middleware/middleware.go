package middleware

import (
	"net/http"
	"strings"

	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"github.com/gin-gonic/gin"
)

const (
	ctxUserID  = "user_id"
	ctxIsAdmin = "is_admin"
)

func Auth(userClient userv1.UserServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")

		resp, err := userClient.ValidateToken(c.Request.Context(), &userv1.ValidateTokenRequest{
			AccessToken: token,
		})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set(ctxUserID, resp.UserId)
		c.Next()
	}
}

// Admin checks that the caller's email is in the allowlist.
// Must be chained after Auth.
func Admin(userClient userv1.UserServiceClient, adminEmails []string) gin.HandlerFunc {
	allow := make(map[string]struct{}, len(adminEmails))
	for _, e := range adminEmails {
		allow[strings.ToLower(strings.TrimSpace(e))] = struct{}{}
	}
	return func(c *gin.Context) {
		uid := CallerID(c)
		if uid == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if len(allow) == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin disabled"})
			return
		}
		resp, err := userClient.GetProfile(c.Request.Context(), &userv1.GetProfileRequest{UserId: uid})
		if err != nil || resp.User == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if _, ok := allow[strings.ToLower(resp.User.Email)]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Set(ctxIsAdmin, true)
		c.Next()
	}
}

func CallerID(c *gin.Context) string {
	id, _ := c.Get(ctxUserID)
	s, _ := id.(string)
	return s
}

func IsAdmin(c *gin.Context) bool {
	v, ok := c.Get(ctxIsAdmin)
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}
