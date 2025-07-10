package repository

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

// TokenRepository defines operations for blacklisting and checking JWT IDs.
type TokenRepository interface {
	// Add stores or updates a revoked token’s JTI and expiration.
	Add(token *model.BlacklistedToken) error

	// IsBlacklisted returns true if the given JTI exists and is not yet expired.
	IsBlacklisted(jti string) (bool, error)

	// RemoveExpired deletes all tokens whose expiration has passed.
	RemoveExpired() error
}

type tokenRepo struct {
	db *gorm.DB
}

// NewTokenRepo creates a new GORM-backed TokenRepository.
func NewTokenRepo(db *gorm.DB) TokenRepository {
	return &tokenRepo{db: db}
}

func (r *tokenRepo) Add(token *model.BlacklistedToken) error {
	return r.db.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "jti"}},
			DoUpdates: clause.AssignmentColumns([]string{"expires_at"}),
		}).
		Create(token).
		Error
}

func (r *tokenRepo) IsBlacklisted(jti string) (bool, error) {
	var count int64
	err := r.db.
		Model(&model.BlacklistedToken{}).
		Where("jti = ? AND expires_at > ?", jti, time.Now()).
		Count(&count).
		Error
	return count > 0, err
}

func (r *tokenRepo) RemoveExpired() error {
	return r.db.
		Where("expires_at < ?", time.Now()).
		Delete(&model.BlacklistedToken{}).
		Error
}
