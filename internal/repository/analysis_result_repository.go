package repository

import (
	"gorm.io/gorm"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

// AnalysisResultRepository defines DB ops for analysis results.
type AnalysisResultRepository interface {
	Create(ar *model.AnalysisResult) error
	ListByURL(urlID uint, p Pagination) ([]model.AnalysisResult, error)
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

func (r *analysisResultRepo) ListByURL(urlID uint, p Pagination) ([]model.AnalysisResult, error) {
	var results []model.AnalysisResult
	err := r.db.
		Where("url_id = ?", urlID).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&results).Error
	return results, err
}
