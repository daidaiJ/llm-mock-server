package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func Auth(authKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authKey == "" {
			c.Next()
			return
		}

		// Check Authorization: Bearer <key>
		if key := extractBearerToken(c.GetHeader("Authorization")); key == authKey {
			c.Next()
			return
		}

		// Check x-api-key: <key> (Anthropic-style)
		if c.GetHeader("x-api-key") == authKey {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "Invalid API key",
				"type":    "authentication_error",
			},
		})
	}
}

func extractBearerToken(authHeader string) string {
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}
