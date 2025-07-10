package repository_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/tests/integration"
)

// Helper function to count tokens in the database
func countTokens(t *testing.T, db *gorm.DB) int64 {
	var count int64
	err := db.Model(&model.BlacklistedToken{}).Count(&count).Error
	require.NoError(t, err, "Should count tokens without error")
	return count
}

func TestBlacklistedTokenRepo_Integration(t *testing.T) {
	// Get a clean database state
	db := integration.SetupTest(t)

	// Create the token repository
	tokenRepo := repository.NewBlacklistedTokenRepo(db)

	// Test data
	testToken := &model.BlacklistedToken{
		JTI:       "test-jwt-id-123",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	t.Run("Create", func(t *testing.T) {
		err := tokenRepo.Create(testToken)
		require.NoError(t, err, "Should create token without error")
		assert.NotZero(t, testToken.ID, "Token ID should be set after creation")
		assert.False(t, testToken.CreatedAt.IsZero(), "CreatedAt should be set")
	})

	t.Run("Exists", func(t *testing.T) {
		// Check that the token we just created exists
		exists, err := tokenRepo.Exists(testToken.JTI)
		require.NoError(t, err, "Should check existence without error")
		assert.True(t, exists, "Token should exist in the blacklist")

		// Check that a non-existent token doesn't exist
		exists, err = tokenRepo.Exists("non-existent-token")
		require.NoError(t, err, "Should check non-existence without error")
		assert.False(t, exists, "Non-existent token should not exist in the blacklist")
	})

	t.Run("Delete Expired", func(t *testing.T) {
		// Create an expired token (expiry time in the past)
		expiredToken := &model.BlacklistedToken{
			JTI:       "expired-token-123",
			ExpiresAt: time.Now().Add(-24 * time.Hour), // Yesterday
		}
		err := tokenRepo.Create(expiredToken)
		require.NoError(t, err, "Should create expired token")

		// Create another non-expired token
		futureToken := &model.BlacklistedToken{
			JTI:       "future-token-456",
			ExpiresAt: time.Now().Add(48 * time.Hour), // 2 days from now
		}
		err = tokenRepo.Create(futureToken)
		require.NoError(t, err, "Should create future token")

		// Verify all tokens exist before deletion
		count := countTokens(t, db)
		assert.Equal(t, int64(3), count, "Should have 3 tokens before deletion")

		// Delete expired tokens
		err = tokenRepo.DeleteExpired()
		require.NoError(t, err, "Should delete expired tokens without error")

		// Verify only the expired token was deleted
		count = countTokens(t, db)
		assert.Equal(t, int64(2), count, "Should have 2 tokens after deletion")

		// Verify the expired token no longer exists
		exists, err := tokenRepo.Exists(expiredToken.JTI)
		require.NoError(t, err, "Should check expired token existence without error")
		assert.False(t, exists, "Expired token should not exist after deletion")

		// Verify the future token still exists
		exists, err = tokenRepo.Exists(futureToken.JTI)
		require.NoError(t, err, "Should check future token existence without error")
		assert.True(t, exists, "Future token should still exist after deletion")

		// Verify the original token still exists
		exists, err = tokenRepo.Exists(testToken.JTI)
		require.NoError(t, err, "Should check original token existence without error")
		assert.True(t, exists, "Original token should still exist after deletion")
	})

	integration.CleanTestData(t)
}
