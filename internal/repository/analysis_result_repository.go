package repository

import (
	"gorm.io/gorm"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

type AnalysisResultRepository interface {
	Create(res *model.AnalysisResult, links []model.Link) error
	ListByURL(urlID uint, p Pagination) ([]model.AnalysisResult, error)
}

type analysisResultRepo struct{ db *gorm.DB }

func NewAnalysisResultRepo(db *gorm.DB) AnalysisResultRepository {
	return &analysisResultRepo{db: db}
}

func (r *analysisResultRepo) Create(res *model.AnalysisResult, links []model.Link) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(res).Error; err != nil {
			return err
		}
		for i := range links {
			links[i].URLID = res.URLID
		}
		return tx.CreateInBatches(&links, 500).Error
	})
}

func (r *analysisResultRepo) ListByURL(urlID uint, p Pagination) ([]model.AnalysisResult, error) {
	var results []model.AnalysisResult
	err := r.db.
		Where("url_id = ?", urlID).
		Order("created_at DESC").
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&results).Error
	return results, err
}
