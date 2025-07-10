package service

import (
	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// AnalysisService manages creation and retrieval of analysis results.
type AnalysisService interface {
	Record(ar *model.AnalysisResult) error
	List(urlID uint, p repository.Pagination) ([]*model.AnalysisResultDTO, error)
}

type analysisService struct {
	repo repository.AnalysisResultRepository
}

// NewAnalysisService constructs an AnalysisService.
func NewAnalysisService(repo repository.AnalysisResultRepository) AnalysisService {
	return &analysisService{repo: repo}
}

func (s *analysisService) Record(ar *model.AnalysisResult) error {
	return s.repo.Create(ar)
}

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
