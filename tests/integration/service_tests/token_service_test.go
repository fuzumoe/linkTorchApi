package service_test

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
	"github.com/fuzumoe/urlinsight-backend/tests/integration"
)

func TestTokenService_Integration(t *testing.T) {
	// Setup test database.
	db := integration.SetupTest(t)
	defer integration.CleanTestData(t)

	// Initialize repository and service with real DB.
	tokenRepo := repository.NewTokenRepo(db)
	jwtSecret := "integration-test-secret-key"
	tokenLifetime := 1 * time.Hour
	tokenService := service.NewTokenService(jwtSecret, tokenLifetime, tokenRepo)

	// Test user ID for token claims
	userID := uint(12345)

	// Clean the blacklisted tokens table before each subtest.
	cleanBlacklistedTokens := func() {
		db.Exec("DELETE FROM blacklisted_tokens")
	}

	t.Run("TokenLifecycle", func(t *testing.T) {
		cleanBlacklistedTokens()

		//  Generate token.
		token, err := tokenService.Generate(userID)
		require.NoError(t, err, "Token generation should succeed")
		require.NotEmpty(t, token, "Generated token should not be empty")

		//  Validate token.
		claims, err := tokenService.Validate(token)
		require.NoError(t, err, "Token validation should succeed")
		assert.Equal(t, userID, claims.UserID, "Token should contain the correct user ID")
		assert.NotEmpty(t, claims.Id, "Token should have a JTI")

		// Extract JTI and expiry for later use.
		jti := claims.Id
		expiresAt := time.Unix(claims.ExpiresAt, 0)

		//  Explicitly blacklist the token.
		err = tokenService.Blacklist(jti, expiresAt)
		require.NoError(t, err, "Blacklisting the token should succeed")

		// Verify token is now blacklisted.
		isBlacklisted, err := tokenService.IsBlacklisted(jti)
		require.NoError(t, err, "Checking blacklist status should succeed")
		assert.True(t, isBlacklisted, "Token should be blacklisted after explicit blacklisting")
	})

	t.Run("InvalidToken", func(t *testing.T) {
		cleanBlacklistedTokens()

		// Test validation of invalid token string.
		claims, err := tokenService.Validate("invalid.token.string")
		assert.Error(t, err, "Validating invalid token should fail")
		assert.Nil(t, claims, "Claims should be nil for invalid token")
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		cleanBlacklistedTokens()

		// Create a token with explicit expiration in the past.
		expiredClaims := model.TokenClaims{
			UserID: userID,
			StandardClaims: jwt.StandardClaims{
				Id:        model.NewJTI(),
				IssuedAt:  time.Now().Add(-2 * time.Hour).Unix(),
				ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
		expiredToken, err := token.SignedString([]byte(jwtSecret))
		require.NoError(t, err, "Creating expired token should succeed")

		// Try to validate the expired token.
		claims, err := tokenService.Validate(expiredToken)
		assert.Error(t, err, "Validating expired token should fail")
		assert.Nil(t, claims, "Claims should be nil for expired token")
	})

	t.Run("TokenWithDifferentSignature", func(t *testing.T) {
		cleanBlacklistedTokens()

		// Create a token with a different signature.
		differentSecretService := service.NewTokenService("different-secret", tokenLifetime, tokenRepo)
		token, err := differentSecretService.Generate(userID)
		require.NoError(t, err, "Token generation should succeed")

		// Try to validate with the original service.
		claims, err := tokenService.Validate(token)
		assert.Error(t, err, "Validating token with wrong signature should fail")
		assert.Nil(t, claims, "Claims should be nil for token with wrong signature")
	})

	t.Run("MultipleTokenBlacklisting", func(t *testing.T) {
		cleanBlacklistedTokens()

		// Generate multiple tokens.
		token1, err := tokenService.Generate(userID)
		require.NoError(t, err)
		token2, err := tokenService.Generate(userID)
		require.NoError(t, err)

		// Validate and extract JTIs.
		claims1, err := tokenService.Validate(token1)
		require.NoError(t, err)
		jti1 := claims1.Id
		expiresAt1 := time.Unix(claims1.ExpiresAt, 0)

		claims2, err := tokenService.Validate(token2)
		require.NoError(t, err)
		jti2 := claims2.Id
		expiresAt2 := time.Unix(claims2.ExpiresAt, 0)
		err = tokenService.Blacklist(jti1, expiresAt1)
		require.NoError(t, err)

		// Verify first token is blacklisted.
		isBlacklisted1, err := tokenService.IsBlacklisted(jti1)
		require.NoError(t, err)
		assert.True(t, isBlacklisted1, "First token should be blacklisted after explicit blacklisting")

		// For token2, we expect it to be stored (and hence blacklisted) already.
		isBlacklisted2, err := tokenService.IsBlacklisted(jti2)
		require.NoError(t, err)
		assert.True(t, isBlacklisted2, "Second token is stored and therefore blacklisted by default")

		err = tokenService.Blacklist(jti2, expiresAt2)
		require.NoError(t, err)
		isBlacklisted2, err = tokenService.IsBlacklisted(jti2)
		require.NoError(t, err)
		assert.True(t, isBlacklisted2, "Second token should remain blacklisted after explicit blacklisting")
	})
}
