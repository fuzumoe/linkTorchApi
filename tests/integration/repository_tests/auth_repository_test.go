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

func countTokens(t *testing.T, db *gorm.DB) int64 {
	var count int64
	err := db.Model(&model.BlacklistedToken{}).Count(&count).Error
	require.NoError(t, err, "Should count tokens without error")
	return count
}

func TestTokenRepo_Integration(t *testing.T) {

	db := utils.SetupTest(t)

	tokenRepo := repository.NewTokenRepo(db)

	testToken := &model.BlacklistedToken{
		JTI:       "test-jwt-id-123",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	t.Run("Add", func(t *testing.T) {
		err := tokenRepo.Add(testToken)
		require.NoError(t, err, "Should add token without error")

		var savedToken model.BlacklistedToken
		err = db.Where("jti = ?", testToken.JTI).First(&savedToken).Error
		require.NoError(t, err, "Should be able to find the token in database")
		assert.Equal(t, testToken.JTI, savedToken.JTI)
		assert.False(t, savedToken.CreatedAt.IsZero(), "CreatedAt should be set")
	})

	t.Run("IsBlacklisted", func(t *testing.T) {

		isBlacklisted, err := tokenRepo.IsBlacklisted(testToken.JTI)
		require.NoError(t, err, "Should check blacklist status without error")
		assert.True(t, isBlacklisted, "Token should be in the blacklist")

		isBlacklisted, err = tokenRepo.IsBlacklisted("non-existent-token")
		require.NoError(t, err, "Should check non-existence without error")
		assert.False(t, isBlacklisted, "Non-existent token should not be in the blacklist")
	})

	t.Run("Add Upsert", func(t *testing.T) {

		updatedToken := &model.BlacklistedToken{
			JTI:       testToken.JTI,
			ExpiresAt: time.Now().Add(72 * time.Hour),
		}

		err := tokenRepo.Add(updatedToken)
		require.NoError(t, err, "Should update token without error")

		var savedToken model.BlacklistedToken
		err = db.Where("jti = ?", testToken.JTI).First(&savedToken).Error
		require.NoError(t, err)
		assert.WithinDuration(t, updatedToken.ExpiresAt, savedToken.ExpiresAt, time.Second)

		count := countTokens(t, db)
		assert.Equal(t, int64(1), count, "Should still have only 1 token after update")
	})

	t.Run("RemoveExpired", func(t *testing.T) {

		expiredToken := &model.BlacklistedToken{
			JTI:       "expired-token-123",
			ExpiresAt: time.Now().Add(-24 * time.Hour),
		}
		err := tokenRepo.Add(expiredToken)
		require.NoError(t, err, "Should create expired token")

		futureToken := &model.BlacklistedToken{
			JTI:       "future-token-456",
			ExpiresAt: time.Now().Add(48 * time.Hour),
		}
		err = tokenRepo.Add(futureToken)
		require.NoError(t, err, "Should create future token")

		count := countTokens(t, db)
		assert.Equal(t, int64(3), count, "Should have 3 tokens before deletion")

		err = tokenRepo.RemoveExpired()
		require.NoError(t, err, "Should remove expired tokens without error")

		count = countTokens(t, db)
		assert.Equal(t, int64(2), count, "Should have 2 tokens after deletion")

		isBlacklisted, err := tokenRepo.IsBlacklisted(expiredToken.JTI)
		require.NoError(t, err, "Should check expired token blacklist status without error")
		assert.False(t, isBlacklisted, "Expired token should not be blacklisted after removal")

		isBlacklisted, err = tokenRepo.IsBlacklisted(futureToken.JTI)
		require.NoError(t, err, "Should check future token blacklist status without error")
		assert.True(t, isBlacklisted, "Future token should still be blacklisted after removal")

		isBlacklisted, err = tokenRepo.IsBlacklisted(testToken.JTI)
		require.NoError(t, err, "Should check original token blacklist status without error")
		assert.True(t, isBlacklisted, "Original token should still be blacklisted after removal")
	})

	utils.CleanTestData(t)
}
