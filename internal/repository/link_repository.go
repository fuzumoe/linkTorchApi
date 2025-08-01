package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

type LinkRepository interface {
	Create(link *model.Link) error
	ListByURL(urlID uint, p Pagination) ([]model.Link, error)
	CountByURL(urlID uint) (int, error)
	Update(link *model.Link) error
	Delete(link *model.Link) error
}

type linkRepo struct {
	db *gorm.DB
}

func (r *linkRepo) CountByURL(urlID uint) (int, error) {
	var count int64
	err := r.db.Model(&model.Link{}).Where("url_id = ?", urlID).Count(&count).Error
	return int(count), err
}

func NewLinkRepo(db *gorm.DB) LinkRepository {
	return &linkRepo{db: db}
}

func (r *linkRepo) Create(link *model.Link) error {
	return r.db.Create(link).Error
}

func (r *linkRepo) ListByURL(urlID uint, p Pagination) ([]model.Link, error) {
	var links []model.Link
	err := r.db.
		Where("url_id = ?", urlID).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&links).Error
	return links, err
}

func (r *linkRepo) Update(link *model.Link) error {
	return r.db.Save(link).Error
}

func (r *linkRepo) Delete(link *model.Link) error {
	result := r.db.Delete(link)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("link not found")
	}
	return nil
}
