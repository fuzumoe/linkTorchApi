package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

// URLRepository defines DB ops around URL entities.
type URLRepository interface {
	Create(u *model.URL) error
	FindByID(id uint) (*model.URL, error)
	ListByUser(userID uint) ([]model.URL, error)
	Update(u *model.URL) error
	Delete(id uint) error
}

type urlRepo struct {
	db *gorm.DB
}

func NewURLRepo(db *gorm.DB) URLRepository {
	return &urlRepo{db: db}
}

func (r *urlRepo) Create(u *model.URL) error {
	return r.db.Create(u).Error
}

func (r *urlRepo) FindByID(id uint) (*model.URL, error) {
	var u model.URL
	if err := r.db.
		Preload("AnalysisResults").
		Preload("Links").
		First(&u, id).
		Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *urlRepo) ListByUser(userID uint) ([]model.URL, error) {
	var urls []model.URL
	if err := r.db.Where("user_id = ?", userID).Find(&urls).Error; err != nil {
		return nil, err
	}
	return urls, nil
}

func (r *urlRepo) Update(u *model.URL) error {
	return r.db.Save(u).Error
}

func (r *urlRepo) Delete(id uint) error {
	res := r.db.Delete(&model.URL{}, id)
	if res.RowsAffected == 0 {
		return errors.New("url not found")
	}
	return res.Error
}
