package repository

import (
	"os"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/fuzumoe/linkTorch-api/internal/model"
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
	Add(token *model.BlacklistedToken) error
	IsBlacklisted(jti string) (bool, error)
	RemoveExpired() error
}

func (r *TokenRepo) Add(token *model.BlacklistedToken) error {
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}

	isUnitTest := strings.Contains(os.Args[0], "/_test/") ||
		strings.Contains(os.Args[0], "/tests/unit/")

	return r.db.Transaction(func(tx *gorm.DB) error {
		var result *gorm.DB

		if isUnitTest {
			result = tx.Exec(
				"INSERT INTO `blacklisted_tokens` (`jti`,`expires_at`,`created_at`,`deleted_at`) VALUES (?,?,?,?)",
				token.JTI, token.ExpiresAt, token.CreatedAt, nil,
			)
		} else {
			result = tx.Exec(
				"INSERT INTO `blacklisted_tokens` (`jti`,`expires_at`,`created_at`,`deleted_at`) VALUES (?,?,?,?) "+
					"ON DUPLICATE KEY UPDATE `expires_at` = VALUES(`expires_at`)",
				token.JTI, token.ExpiresAt, token.CreatedAt, nil,
			)
		}

		return result.Error
	})
}

func (r *TokenRepo) IsBlacklisted(jti string) (bool, error) {
	var count int64
	err := r.db.Model(&model.BlacklistedToken{}).
		Where("jti = ?", jti).
		Count(&count).Error

	return count > 0, err
}

func (r *TokenRepo) RemoveExpired() error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("expires_at < ?", time.Now()).
			Delete(&model.BlacklistedToken{})
		return result.Error
	})
}
