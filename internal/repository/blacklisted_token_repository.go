package repository

import (
	"gorm.io/gorm"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

// BlacklistedTokenRepository defines DB ops for token blacklisting.
type BlacklistedTokenRepository interface {
	Create(token *model.BlacklistedToken) error
	Exists(jti string) (bool, error)
	DeleteExpired() error
}

type blacklistedTokenRepo struct {
	db *gorm.DB
}

func NewBlacklistedTokenRepo(db *gorm.DB) BlacklistedTokenRepository {
	return &blacklistedTokenRepo{db: db}
}

func (r *blacklistedTokenRepo) Create(token *model.BlacklistedToken) error {
	return r.db.Create(token).Error
}

func (r *blacklistedTokenRepo) Exists(jti string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.BlacklistedToken{}).Where("jti = ?", jti).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *blacklistedTokenRepo) DeleteExpired() error {
	res := r.db.Where("expires_at < NOW()").Delete(&model.BlacklistedToken{})
	if res.Error != nil && res.Error != gorm.ErrRecordNotFound {
		return res.Error
	}
	return nil
}
