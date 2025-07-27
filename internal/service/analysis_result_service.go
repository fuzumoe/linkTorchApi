package service

import (
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

type AnalysisService interface {
	Record(res *model.AnalysisResult, links []model.Link) error
	List(urlID uint, p repository.Pagination) ([]*model.AnalysisResultDTO, error)
}

type analysisService struct {
	repo repository.AnalysisResultRepository
}

func NewAnalysisService(r repository.AnalysisResultRepository) AnalysisService {
	return &analysisService{repo: r}
}

func (s *analysisService) Record(res *model.AnalysisResult, links []model.Link) error {
	return s.repo.Create(res, links)
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
