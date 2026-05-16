package middleware

import (
	"net/http"
	"strings"

	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"github.com/gin-gonic/gin"
)

const ctxUserID = "user_id"

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

func CallerID(c *gin.Context) string {
	id, _ := c.Get(ctxUserID)
	s, _ := id.(string)
	return s
}
