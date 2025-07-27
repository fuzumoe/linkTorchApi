package repository

import (
	"gorm.io/gorm"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

// AnalysisResultRepository defines DB ops for analysis results.
type AnalysisResultRepository interface {
	Create(res *model.AnalysisResult, links []model.Link) error
	ListByURL(urlID uint, p Pagination) ([]model.AnalysisResult, error)
}

type analysisResultRepo struct{ db *gorm.DB }

func NewAnalysisResultRepo(db *gorm.DB) AnalysisResultRepository {
	return &analysisResultRepo{db: db}
}

// Create stores one AnalysisResult plus all its Link rows in a single TX.
func (r *analysisResultRepo) Create(res *model.AnalysisResult, links []model.Link) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(res).Error; err != nil {
			return err
		}
		for i := range links {
			links[i].URLID = res.URLID // foreign key
		}
		return tx.CreateInBatches(&links, 500).Error
	})
}

// ListByURL returns paginated snapshots, newest first.
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
