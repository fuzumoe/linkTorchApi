package service

import (
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

// AnalysisService manages creation and retrieval of analysis results.
type AnalysisService interface {
	// Record persists one AnalysisResult and all its Link rows atomically.
	Record(res *model.AnalysisResult, links []model.Link) error

	// List returns paginated snapshots for a URL, newest first.
	List(urlID uint, p repository.Pagination) ([]*model.AnalysisResultDTO, error)
}

type analysisService struct {
	repo repository.AnalysisResultRepository
}

// NewAnalysisService constructs an AnalysisService.
func NewAnalysisService(r repository.AnalysisResultRepository) AnalysisService {
	return &analysisService{repo: r}
}

// Record delegates to repo.Create, which stores result + links in one TX.
func (s *analysisService) Record(res *model.AnalysisResult, links []model.Link) error {
	return s.repo.Create(res, links)
}

// List converts the repo models to DTOs and applies Pagination.
func (s *analysisService) List(urlID uint, p repository.Pagination) ([]*model.AnalysisResultDTO, error) {
	results, err := s.repo.ListByURL(urlID, p)
	if err != nil {
		return nil, err
	}
	dtos := make([]*model.AnalysisResultDTO, len(results))
	for i, r := range results {
		dtos[i] = r.ToDTO()
	}
	return dtos, nil
}
