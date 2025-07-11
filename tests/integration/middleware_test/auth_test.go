package middleware_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/middleware"
	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
	"github.com/fuzumoe/urlinsight-backend/tests/integration"
)

func TestBasicAuthMiddleware_Integration(t *testing.T) {
	t.Run("BasicAuth success", func(t *testing.T) {
		// Setup integration database.
		db := integration.SetupTest(t)
		defer integration.CleanTestData(t)

		// Create user repository and service.
		userRepo := repository.NewUserRepo(db)
		// Assume NewUserService implements service.UserService.
		userService := service.NewUserService(userRepo)

		// Register a test user.
		createInput := &model.CreateUserInput{
			Email:    "user@example.com",
			Password: "correctpassword",
			Username: "testuser",
		}
		registeredUser, err := userService.Register(createInput)
		require.NoError(t, err)
		require.NotZero(t, registeredUser.ID)

		// Setup Gin with the BasicAuthMiddleware.
		router := gin.New()
		router.Use(middleware.BasicAuthMiddleware(userService))
		router.GET("/protected", func(c *gin.Context) {
			c.String(http.StatusOK, "basic auth passed")
		})

		// Prepare a valid Basic Auth header.
		cred := base64.StdEncoding.EncodeToString([]byte("user@example.com:correctpassword"))
		req, err := http.NewRequest("GET", "/protected", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Basic "+cred)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "basic auth passed", w.Body.String())
	})
}

func TestJWTAuthMiddleware_Integration(t *testing.T) {
	t.Run("JWTAuth success", func(t *testing.T) {
		// Setup integration database.
		db := integration.SetupTest(t)
		defer integration.CleanTestData(t)

		// Create repositories.
		userRepo := repository.NewUserRepo(db)
		tokenRepo := repository.NewTokenRepo(db)
		// Create authService with required arguments:
		authService := service.NewAuthService(userRepo, tokenRepo, "secret", time.Hour)
		// Use userRepo as the user lookup.
		userService := service.NewUserService(userRepo)

		// Register a test user.
		createInput := &model.CreateUserInput{
			Email:    "user@example.com",
			Password: "correctpassword",
			Username: "jwtuser",
		}
		registeredUser, err := userService.Register(createInput)
		require.NoError(t, err)
		require.NotZero(t, registeredUser.ID)

		// Generate a JWT token for the user.
		tokenString, err := authService.Generate(registeredUser.ID)
		require.NoError(t, err)
		require.NotEmpty(t, tokenString)

		// Setup Gin with the JWTAuthMiddleware.
		router := gin.New()
		router.Use(middleware.JWTAuthMiddleware(authService, userRepo))
		router.GET("/jwt-protected", func(c *gin.Context) {
			c.String(http.StatusOK, "jwt auth passed")
		})

		// Create a valid JWT header.
		req, err := http.NewRequest("GET", "/jwt-protected", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "jwt auth passed", w.Body.String())
	})
}

func TestJWTAuthMiddleware_RevokedToken_Integration(t *testing.T) {
	t.Run("JWTAuth revoked token returns unauthorized", func(t *testing.T) {
		// Setup integration database.
		db := integration.SetupTest(t)
		defer integration.CleanTestData(t)

		userRepo := repository.NewUserRepo(db)
		tokenRepo := repository.NewTokenRepo(db)
		// Create authService with required arguments.
		authService := service.NewAuthService(userRepo, tokenRepo, "secret", time.Hour)
		userService := service.NewUserService(userRepo)

		// Register a test user.
		createInput := &model.CreateUserInput{
			Email:    "user@example.com",
			Password: "correctpassword",
			Username: "revokeduser",
		}
		registeredUser, err := userService.Register(createInput)
		require.NoError(t, err)
		require.NotZero(t, registeredUser.ID)

		// Generate a token.
		tokenString, err := authService.Generate(registeredUser.ID)
		require.NoError(t, err)
		require.NotEmpty(t, tokenString)

		// Manually parse the token to extract claims.
		claims := &jwt.RegisteredClaims{}
		_, err = jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		})
		require.NoError(t, err)
		// Assuming model.BlacklistedToken has fields JTI and ExpiredAt.
		blToken := &model.BlacklistedToken{
			JTI:       claims.ID,
			ExpiresAt: claims.ExpiresAt.Time, // token's expiration
		}
		err = tokenRepo.Add(blToken)
		require.NoError(t, err)

		// Setup Gin with the JWTAuthMiddleware.
		router := gin.New()
		router.Use(middleware.JWTAuthMiddleware(authService, userRepo))
		router.GET("/revoked", func(c *gin.Context) {
			c.String(http.StatusOK, "should not pass")
		})

		req, err := http.NewRequest("GET", "/revoked", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		// Expect unauthorized since the token has been blacklisted.
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
