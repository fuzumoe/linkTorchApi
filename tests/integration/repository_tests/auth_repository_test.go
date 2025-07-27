package repository_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

// Helper function to count tokens in the database.
func countTokens(t *testing.T, db *gorm.DB) int64 {
	var count int64
	err := db.Model(&model.BlacklistedToken{}).Count(&count).Error
	require.NoError(t, err, "Should count tokens without error")
	return count
}

func TestTokenRepo_Integration(t *testing.T) {
	// Get a clean database state.
	db := utils.SetupTest(t)

	// Create the token repository.
	tokenRepo := repository.NewTokenRepo(db)

	// Test data.
	testToken := &model.BlacklistedToken{
		JTI:       "test-jwt-id-123",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	t.Run("Add", func(t *testing.T) {
		err := tokenRepo.Add(testToken)
		require.NoError(t, err, "Should add token without error")

		// Check that we can retrieve the token from DB directly to verify it was saved.
		var savedToken model.BlacklistedToken
		err = db.Where("jti = ?", testToken.JTI).First(&savedToken).Error
		require.NoError(t, err, "Should be able to find the token in database")
		assert.Equal(t, testToken.JTI, savedToken.JTI)
		assert.False(t, savedToken.CreatedAt.IsZero(), "CreatedAt should be set")
	})

	t.Run("IsBlacklisted", func(t *testing.T) {
		// Check that the token we just added is blacklisted.
		isBlacklisted, err := tokenRepo.IsBlacklisted(testToken.JTI)
		require.NoError(t, err, "Should check blacklist status without error")
		assert.True(t, isBlacklisted, "Token should be in the blacklist")

		// Check that a non-existent token isn't blacklisted.
		isBlacklisted, err = tokenRepo.IsBlacklisted("non-existent-token")
		require.NoError(t, err, "Should check non-existence without error")
		assert.False(t, isBlacklisted, "Non-existent token should not be in the blacklist")
	})

	t.Run("Add Upsert", func(t *testing.T) {
		// Create a token with the same JTI but different expiration.
		updatedToken := &model.BlacklistedToken{
			JTI:       testToken.JTI,
			ExpiresAt: time.Now().Add(72 * time.Hour),
		}

		// Add should update the existing token
		err := tokenRepo.Add(updatedToken)
		require.NoError(t, err, "Should update token without error")

		// Verify the token was updated by checking the expiration.
		var savedToken model.BlacklistedToken
		err = db.Where("jti = ?", testToken.JTI).First(&savedToken).Error
		require.NoError(t, err)
		assert.WithinDuration(t, updatedToken.ExpiresAt, savedToken.ExpiresAt, time.Second)

		// Count should still be 1 since we updated, not inserted.
		count := countTokens(t, db)
		assert.Equal(t, int64(1), count, "Should still have only 1 token after update")
	})

	t.Run("RemoveExpired", func(t *testing.T) {
		// Create an expired token.
		expiredToken := &model.BlacklistedToken{
			JTI:       "expired-token-123",
			ExpiresAt: time.Now().Add(-24 * time.Hour), // Yesterday.
		}
		err := tokenRepo.Add(expiredToken)
		require.NoError(t, err, "Should create expired token")

		// Create another non-expired token
		futureToken := &model.BlacklistedToken{
			JTI:       "future-token-456",
			ExpiresAt: time.Now().Add(48 * time.Hour), // 2 days from now.
		}
		err = tokenRepo.Add(futureToken)
		require.NoError(t, err, "Should create future token")

		// Verify all tokens exist before deletion.
		count := countTokens(t, db)
		assert.Equal(t, int64(3), count, "Should have 3 tokens before deletion")

		// Remove expired tokens.
		err = tokenRepo.RemoveExpired()
		require.NoError(t, err, "Should remove expired tokens without error")

		// Verify only the expired token was deleted.
		count = countTokens(t, db)
		assert.Equal(t, int64(2), count, "Should have 2 tokens after deletion")

		// Verify the expired token no longer exists.
		isBlacklisted, err := tokenRepo.IsBlacklisted(expiredToken.JTI)
		require.NoError(t, err, "Should check expired token blacklist status without error")
		assert.False(t, isBlacklisted, "Expired token should not be blacklisted after removal")

		// Verify the future token still exists.
		isBlacklisted, err = tokenRepo.IsBlacklisted(futureToken.JTI)
		require.NoError(t, err, "Should check future token blacklist status without error")
		assert.True(t, isBlacklisted, "Future token should still be blacklisted after removal")

		// Verify the original token still exists.
		isBlacklisted, err = tokenRepo.IsBlacklisted(testToken.JTI)
		require.NoError(t, err, "Should check original token blacklist status without error")
		assert.True(t, isBlacklisted, "Original token should still be blacklisted after removal")
	})

	utils.CleanTestData(t)
}
