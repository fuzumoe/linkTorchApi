package repository

import (
	"gorm.io/gorm"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

// AnalysisResultRepository defines DB ops for analysis results.
type AnalysisResultRepository interface {
	Create(ar *model.AnalysisResult) error
	ListByURL(urlID uint) ([]model.AnalysisResult, error)
}

type analysisResultRepo struct {
	db *gorm.DB
}

func NewAnalysisResultRepo(db *gorm.DB) AnalysisResultRepository {
	return &analysisResultRepo{db: db}
}

func (r *analysisResultRepo) Create(ar *model.AnalysisResult) error {
	return r.db.Create(ar).Error
}

func (r *analysisResultRepo) ListByURL(urlID uint) ([]model.AnalysisResult, error) {
	var results []model.AnalysisResult
	if err := r.db.Where("url_id = ?", urlID).Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}
