package middleware

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fuzumoe/urlinsight-backend/internal/service"
)

// BasicAuthMiddleware returns middleware that enables HTTP Basic Auth.
func BasicAuthMiddleware(us service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header missing or not Basic"})
			return
		}
		payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
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
		user, err := us.Authenticate(email, password)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		// store authenticated user ID in context
		c.Set("user_id", user.ID)
		c.Next()
	}
}

// JWTAuthMiddleware returns middleware that enforces Bearer JWT Auth.
func JWTAuthMiddleware(ts service.TokenService, userLookup service.UserLookup) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header missing or not Bearer"})
			return
		}
		tokenString := strings.TrimPrefix(auth, "Bearer ")
		claims, err := ts.Validate(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		// check blacklist
		blacklisted, err := ts.IsBlacklisted(claims.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not verify token"})
			return
		}
		if blacklisted {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
			return
		}
		// optionally load the user to ensure it still exists
		if _, err := userLookup.FindByID(claims.UserID); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user no longer exists"})
			return
		}
		// store user ID and token ID (jti) in context
		c.Set("user_id", claims.UserID)
		c.Set("jti", claims.ID)
		c.Next()
	}
}
