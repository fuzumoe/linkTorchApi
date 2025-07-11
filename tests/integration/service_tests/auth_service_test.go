package service_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
	"github.com/fuzumoe/urlinsight-backend/tests/integration"
)

func TestAuthService_Integration(t *testing.T) {
	// Setup test database.
	db := integration.SetupTest(t)
	defer integration.CleanTestData(t)

	// Initialize repository and service with real DB.
	tokenRepo := repository.NewTokenRepo(db)
	userRepo := repository.NewUserRepo(db)
	jwtSecret := "integration-test-secret-key"
	tokenLifetime := 1 * time.Hour
	authService := service.NewAuthService(userRepo, tokenRepo, jwtSecret, tokenLifetime)

	// Test user ID for token claims
	userID := uint(12345)

	// Create test user in database
	testUser := &model.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}
	err := db.Create(testUser).Error
	require.NoError(t, err, "Test user creation should succeed")

	// Clean the blacklisted tokens table before each subtest.
	cleanBlacklistedTokens := func() {
		db.Exec("DELETE FROM blacklisted_tokens")
	}

	t.Run("TokenLifecycle", func(t *testing.T) {
		cleanBlacklistedTokens()

		//  Generate token.
		token, err := authService.Generate(userID)
		require.NoError(t, err, "Token generation should succeed")
		require.NotEmpty(t, token, "Generated token should not be empty")

		//  Validate token.
		claims, err := authService.Validate(token)
		require.NoError(t, err, "Token validation should succeed")
		assert.Equal(t, userID, claims.UserID, "Token should contain the correct user ID")
		assert.NotEmpty(t, claims.ID, "Token should have a JTI")

		// Extract JTI for later use.
		jti := claims.ID

		//  Explicitly invalidate the token.
		err = authService.Invalidate(jti)
		require.NoError(t, err, "Invalidating the token should succeed")

		// Verify token is now blacklisted.
		isBlacklisted, err := authService.IsBlacklisted(jti)
		require.NoError(t, err, "Checking blacklist status should succeed")
		assert.True(t, isBlacklisted, "Token should be blacklisted after explicit invalidation")
	})

	t.Run("InvalidToken", func(t *testing.T) {
		cleanBlacklistedTokens()

		// Test validation of invalid token string.
		claims, err := authService.Validate("invalid.token.string")
		assert.Error(t, err, "Validating invalid token should fail")
		assert.Equal(t, service.ErrTokenInvalid, err, "Should return token invalid error")
		assert.Nil(t, claims, "Claims should be nil for invalid token")
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		cleanBlacklistedTokens()

		// Create a token with explicit expiration in the past.
		expiredClaims := service.JWTClaims{
			UserID: userID,
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        "test-jti",
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
				Subject:   "access_token",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
		expiredToken, err := token.SignedString([]byte(jwtSecret))
		require.NoError(t, err, "Creating expired token should succeed")

		// Try to validate the expired token.
		claims, err := authService.Validate(expiredToken)
		assert.Error(t, err, "Validating expired token should fail")
		assert.Equal(t, service.ErrTokenExpired, err, "Should return token expired error")
		assert.Nil(t, claims, "Claims should be nil for expired token")
	})

	t.Run("TokenWithDifferentSignature", func(t *testing.T) {
		cleanBlacklistedTokens()

		// Create a token with a different signature.
		differentSecretService := service.NewAuthService(userRepo, tokenRepo, "different-secret", tokenLifetime)
		token, err := differentSecretService.Generate(userID)
		require.NoError(t, err, "Token generation should succeed")

		// Try to validate with the original service.
		claims, err := authService.Validate(token)
		assert.Error(t, err, "Validating token with wrong signature should fail")
		assert.Equal(t, service.ErrTokenInvalid, err, "Should return token invalid error")
		assert.Nil(t, claims, "Claims should be nil for token with wrong signature")
	})

	t.Run("MultipleTokenBlacklisting", func(t *testing.T) {
		cleanBlacklistedTokens()

		// Generate multiple tokens.
		token1, err := authService.Generate(userID)
		require.NoError(t, err)
		token2, err := authService.Generate(userID)
		require.NoError(t, err)

		// Validate and extract JTIs.
		claims1, err := authService.Validate(token1)
		require.NoError(t, err)
		jti1 := claims1.ID

		claims2, err := authService.Validate(token2)
		require.NoError(t, err)
		jti2 := claims2.ID

		err = authService.Invalidate(jti1)
		require.NoError(t, err)

		// Verify first token is blacklisted.
		isBlacklisted1, err := authService.IsBlacklisted(jti1)
		require.NoError(t, err)
		assert.True(t, isBlacklisted1, "First token should be blacklisted after explicit invalidation")

		// Second token should not be blacklisted yet
		isBlacklisted2, err := authService.IsBlacklisted(jti2)
		require.NoError(t, err)
		assert.False(t, isBlacklisted2, "Second token should not be blacklisted yet")

		// Blacklist the second token too
		err = authService.Invalidate(jti2)
		require.NoError(t, err)
		isBlacklisted2, err = authService.IsBlacklisted(jti2)
		require.NoError(t, err)
		assert.True(t, isBlacklisted2, "Second token should be blacklisted after explicit invalidation")
	})

	// NEW: Test user not found case
	t.Run("UserNotFound", func(t *testing.T) {
		cleanBlacklistedTokens()

		// Try to generate token for non-existent user
		nonExistentUserID := uint(99999)
		token, err := authService.Generate(nonExistentUserID)

		assert.Error(t, err, "Token generation should fail for non-existent user")
		assert.Empty(t, token, "Token should be empty for non-existent user")
		assert.Contains(t, err.Error(), "not found", "Error should mention user not found")
	})

	// NEW: Test CleanupExpired
	t.Run("CleanupExpired", func(t *testing.T) {
		cleanBlacklistedTokens()

		// Create an expired token in the database
		expiredToken := &model.BlacklistedToken{
			JTI:       "expired-token-jti",
			ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired 24 hours ago
			CreatedAt: time.Now().Add(-48 * time.Hour), // Created 48 hours ago
		}
		err := tokenRepo.Add(expiredToken)
		require.NoError(t, err, "Adding expired token should succeed")

		// Create a valid token in the database
		validToken := &model.BlacklistedToken{
			JTI:       "valid-token-jti",
			ExpiresAt: time.Now().Add(24 * time.Hour), // Expires 24 hours from now
			CreatedAt: time.Now(),
		}
		err = tokenRepo.Add(validToken)
		require.NoError(t, err, "Adding valid token should succeed")

		// Verify both tokens exist
		var count int64
		err = db.Model(&model.BlacklistedToken{}).Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(2), count, "Should have 2 tokens before cleanup")

		// Run cleanup
		err = authService.CleanupExpired()
		require.NoError(t, err, "Cleanup should succeed")

		// Verify only valid token remains
		err = db.Model(&model.BlacklistedToken{}).Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "Should have 1 token after cleanup")

		// Verify expired token is removed
		isBlacklisted, err := authService.IsBlacklisted(expiredToken.JTI)
		require.NoError(t, err)
		assert.False(t, isBlacklisted, "Expired token should not be blacklisted after cleanup")

		// Verify valid token still exists
		isBlacklisted, err = authService.IsBlacklisted(validToken.JTI)
		require.NoError(t, err)
		assert.True(t, isBlacklisted, "Valid token should still be blacklisted after cleanup")
	})
}
