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
	ListByUser(userID uint, p Pagination) ([]model.URL, error)
	Update(u *model.URL) error
	Delete(id uint) error

	UpdateStatus(id uint, status string) error
	SaveResults(id uint, res *model.AnalysisResult, links []model.Link) error

	Results(id uint) (*model.URL, error)
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

func (r *urlRepo) ListByUser(userID uint, p Pagination) ([]model.URL, error) {
	var urls []model.URL
	err := r.db.
		Where("user_id = ?", userID).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&urls).Error
	return urls, err
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

func (r *urlRepo) UpdateStatus(id uint, status string) error {
	return r.db.
		Model(&model.URL{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *urlRepo) SaveResults(id uint, res *model.AnalysisResult, links []model.Link) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		res.URLID = id
		if err := tx.Create(res).Error; err != nil {
			return err
		}
		for i := range links {
			links[i].URLID = id
		}
		return tx.CreateInBatches(&links, 500).Error
	})
}

func (r *urlRepo) Results(id uint) (*model.URL, error) {
	var u model.URL
	err := r.db.
		Preload("AnalysisResults").
		Preload("Links").
		First(&u, id).Error
	return &u, err
}
