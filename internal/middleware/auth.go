package middleware

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fuzumoe/linkTorch-api/internal/service"
)

func AuthMiddleware(authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header missing"})
			return
		}
		const basicPrefix = "Basic "
		if after, ok := strings.CutPrefix(auth, basicPrefix); ok {
			payload, err := base64.StdEncoding.DecodeString(after)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid base64 credentials"})
				return
			}
			parts := strings.SplitN(string(payload), ":", 2)
			if len(parts) != 2 {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid basic auth format"})
				return
			}
			email, password := parts[0], parts[1]
			user, err := authService.AuthenticateBasic(email, password)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
				return
			}
			c.Set("user_id", user.ID)
			c.Set("user_email", user.Email)
			c.Set("user_role", user.Role)
			c.Next()
			return
		} else if strings.HasPrefix(auth, "Bearer ") {
			tokenString := strings.TrimPrefix(auth, "Bearer ")
			claims, err := authService.Validate(tokenString)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
				return
			}
			revoked, err := authService.IsTokenRevoked(claims.ID)
			if err != nil || revoked {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked or an error occurred"})
				return
			}
			c.Set("user_id", claims.UserID)
			c.Set("user_email", claims.Email)
			c.Set("user_role", claims.Role)
			c.Set("jti", claims.ID)
			c.Next()
			return
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unsupported authorization type"})
			return
		}
	}
}
