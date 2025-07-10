package repository

import (
	"gorm.io/gorm"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

// LinkRepository defines DB ops for Link entities.
type LinkRepository interface {
	Create(link *model.Link) error
	ListByURL(urlID uint) ([]model.Link, error)
	Update(link *model.Link) error
	Delete(link *model.Link) error
}

type linkRepo struct {
	db *gorm.DB
}

func NewLinkRepo(db *gorm.DB) LinkRepository {
	return &linkRepo{db: db}
}

func (r *linkRepo) Create(link *model.Link) error {
	return r.db.Create(link).Error
}

func (r *linkRepo) ListByURL(urlID uint) ([]model.Link, error) {
	var links []model.Link
	if err := r.db.Where("url_id = ?", urlID).Find(&links).Error; err != nil {
		return nil, err
	}
	return links, nil
}

func (r *linkRepo) Update(link *model.Link) error {
	return r.db.Save(link).Error
}

func (r *linkRepo) Delete(link *model.Link) error {
	return r.db.Delete(link).Error
}
