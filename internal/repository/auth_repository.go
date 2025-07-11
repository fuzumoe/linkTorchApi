package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

type TokenRepo struct {
	db *gorm.DB
}

func NewTokenRepo(db *gorm.DB) *TokenRepo {
	return &TokenRepo{
		db: db,
	}
}

type TokenRepository interface {
	// Add adds a token to the blacklist.
	Add(token *model.BlacklistedToken) error
	// IsBlacklisted checks if a token is in the blacklist.
	IsBlacklisted(jti string) (bool, error)
	// RemoveExpired removes expired tokens from the blacklist.
	RemoveExpired() error
}

// Add adds a token to the blacklist or updates its expiry if it already exists
func (r *TokenRepo) Add(token *model.BlacklistedToken) error {
	// Ensure created_at is set
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}

	// Use raw SQL for the upsert operation
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Using raw SQL with ON DUPLICATE KEY UPDATE to handle upsert.
		result := tx.Exec(
			"INSERT INTO `blacklisted_tokens` (`jti`, `expires_at`, `created_at`, `deleted_at`) VALUES (?, ?, ?, NULL) "+
				"ON DUPLICATE KEY UPDATE `expires_at` = VALUES(`expires_at`)",
			token.JTI, token.ExpiresAt, token.CreatedAt,
		)
		return result.Error
	})
}

// IsBlacklisted checks if a token ID is in the blacklist.
func (r *TokenRepo) IsBlacklisted(jti string) (bool, error) {
	var count int64
	err := r.db.Model(&model.BlacklistedToken{}).
		Where("jti = ?", jti).
		Count(&count).Error

	return count > 0, err
}

// RemoveExpired removes expired tokens from the blacklist.
func (r *TokenRepo) RemoveExpired() error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Soft delete tokens that have expired.
		result := tx.Where("expires_at < ?", time.Now()).
			Delete(&model.BlacklistedToken{})
		return result.Error
	})
}
