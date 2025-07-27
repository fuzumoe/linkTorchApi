package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/internal/service"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

func TestAuthService_Integration(t *testing.T) {
	// Set up test database
	db := utils.SetupTest(t)

	// Initialize repositories with real DB
	userRepo := repository.NewUserRepo(db)
	tokenRepo := repository.NewTokenRepo(db)

	// Initialize the auth service
	jwtSecret := "utils-test-secret"
	tokenLifetime := 1 * time.Hour
	authService := service.NewAuthService(userRepo, tokenRepo, jwtSecret, tokenLifetime)

	// Test data
	testUsername := "testuser"
	testEmail := "test@example.com"
	testPassword := "password123"

	// Create a test user for auth tests
	t.Run("Setup_TestUser", func(t *testing.T) {
		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
		require.NoError(t, err)

		// Create user directly through repository
		user := &model.User{
			Username: testUsername,
			Email:    testEmail,
			Password: string(hashedPassword),
		}
		err = userRepo.Create(user)
		require.NoError(t, err)
		require.NotZero(t, user.ID)
	})

	var userID uint
	var tokenString string
	var tokenID string

	t.Run("AuthenticateBasic_Success", func(t *testing.T) {
		// Test basic authentication
		userDTO, err := authService.AuthenticateBasic(testEmail, testPassword)
		require.NoError(t, err)
		assert.NotNil(t, userDTO)
		assert.Equal(t, testUsername, userDTO.Username)
		assert.Equal(t, testEmail, userDTO.Email)

		// Save user ID for later tests
		userID = userDTO.ID
	})

	t.Run("AuthenticateBasic_WrongPassword", func(t *testing.T) {
		userDTO, err := authService.AuthenticateBasic(testEmail, "wrongpassword")
		assert.Error(t, err)
		assert.Nil(t, userDTO)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("AuthenticateBasic_NonExistentUser", func(t *testing.T) {
		userDTO, err := authService.AuthenticateBasic("nonexistent@example.com", testPassword)
		assert.Error(t, err)
		assert.Nil(t, userDTO)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("FindUserById", func(t *testing.T) {
		userDTO, err := authService.FindUserById(userID)
		require.NoError(t, err)
		assert.NotNil(t, userDTO)
		assert.Equal(t, testUsername, userDTO.Username)
		assert.Equal(t, testEmail, userDTO.Email)
	})

	t.Run("FindUserById_NonExistent", func(t *testing.T) {
		userDTO, err := authService.FindUserById(9999) // Non-existent ID
		assert.Error(t, err)
		assert.Nil(t, userDTO)
	})

	t.Run("Generate_Token", func(t *testing.T) {
		// Generate a token for the test user
		token, err := authService.Generate(userID)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Save the token for later tests
		tokenString = token

		// Extract the token ID for revocation tests
		claims, err := authService.Validate(token)
		require.NoError(t, err)
		tokenID = claims.ID
		assert.NotEmpty(t, tokenID)
	})

	t.Run("Generate_NonExistentUser", func(t *testing.T) {
		token, err := authService.Generate(9999) // Non-existent ID
		assert.Error(t, err)
		assert.Empty(t, token)
	})

	t.Run("Validate_ValidToken", func(t *testing.T) {
		claims, err := authService.Validate(tokenString)
		require.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, userID, claims.UserID)
	})

	t.Run("Validate_InvalidToken", func(t *testing.T) {
		claims, err := authService.Validate("invalid.token.string")
		assert.Error(t, err)
		assert.Equal(t, service.ErrTokenInvalid, err)
		assert.Nil(t, claims)
	})

	t.Run("IsTokenRevoked_NotRevoked", func(t *testing.T) {
		// Check that the token is not revoked
		revoked, err := authService.IsTokenRevoked(tokenID)
		require.NoError(t, err)
		assert.False(t, revoked)
	})

	t.Run("Invalidate_Token", func(t *testing.T) {
		// Invalidate the token
		err := authService.Invalidate(tokenID)
		require.NoError(t, err)

		// Verify token is now revoked
		revoked, err := authService.IsTokenRevoked(tokenID)
		require.NoError(t, err)
		assert.True(t, revoked)

		// Validate should now fail
		claims, err := authService.Validate(tokenString)
		assert.Error(t, err)
		assert.Equal(t, service.ErrTokenInvalid, err)
		assert.Nil(t, claims)
	})

	t.Run("Invalidate_EmptyToken", func(t *testing.T) {
		err := authService.Invalidate("")
		assert.Error(t, err)
		assert.Equal(t, service.ErrTokenInvalid, err)
	})

	t.Run("IsTokenRevoked_EmptyToken", func(t *testing.T) {
		revoked, err := authService.IsTokenRevoked("")
		assert.NoError(t, err)
		assert.False(t, revoked)
	})

	t.Run("CleanupExpired", func(t *testing.T) {
		expiredJTI := "expired-token-jti-" + time.Now().Format("20060102150405")
		validJTI := "valid-token-jti-" + time.Now().Format("20060102150405")

		expiredToken := &model.BlacklistedToken{
			JTI:       expiredJTI,
			ExpiresAt: time.Now().Add(-24 * time.Hour),
		}
		err := tokenRepo.Add(expiredToken)
		require.NoError(t, err)

		validToken := &model.BlacklistedToken{
			JTI:       validJTI,
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		err = tokenRepo.Add(validToken)
		require.NoError(t, err)

		// Direct DB check: the expired token should be there.
		var expiredCount int64
		err = db.Model(&model.BlacklistedToken{}).Where("jti = ?", expiredJTI).Count(&expiredCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), expiredCount, "Expired token should be in database")

		// Check via the service.
		isRevoked, err := authService.IsTokenRevoked(expiredJTI)
		require.NoError(t, err)
		assert.True(t, isRevoked, "Expired token should be in blacklist after adding")

		isRevoked, err = authService.IsTokenRevoked(validJTI)
		require.NoError(t, err)
		assert.True(t, isRevoked, "Valid token should be in blacklist after adding")

		// Run cleanup.
		err = authService.CleanupExpired()
		require.NoError(t, err)

		// After cleanup, expired token should be gone.
		err = db.Model(&model.BlacklistedToken{}).Where("jti = ?", expiredJTI).Count(&expiredCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), expiredCount, "Expired token should be removed from database after cleanup")

		var validCount int64
		err = db.Model(&model.BlacklistedToken{}).Where("jti = ?", validJTI).Count(&validCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), validCount, "Valid token should still be in database after cleanup")

		isRevoked, err = authService.IsTokenRevoked(expiredJTI)
		require.NoError(t, err)
		assert.False(t, isRevoked, "Expired token should be removed from blacklist")

		isRevoked, err = authService.IsTokenRevoked(validJTI)
		require.NoError(t, err)
		assert.True(t, isRevoked, "Valid token should remain in blacklist")
	})

	// Clean up test data
	utils.CleanTestData(t)
}
