package service

import (
	"errors"
	"fmt"

	"github.com/fuzumoe/linkTorch-api/internal/crawler"
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

type URLService interface {
	Create(input *model.CreateURLInputDTO) (uint, error)
	Get(id uint) (*model.URLDTO, error)
	List(userID uint, p repository.Pagination) (*model.PaginatedResponse[model.URLDTO], error)
	Update(id uint, input *model.UpdateURLInput) error
	Delete(id uint) error
	Start(id uint) error
	StartWithPriority(id uint, priority int) error
	Stop(id uint) error
	Results(id uint) (*model.URLDTO, error)
	ResultsWithDetails(id uint) (*model.URL, []*model.AnalysisResult, []*model.Link, error)
	GetCrawlResults() <-chan crawler.CrawlResult
	AdjustCrawlerWorkers(action string, count int) error
}

type urlService struct {
	repo     repository.URLRepository
	crawlers crawler.Pool
}

func (s *urlService) Update(id uint, in *model.UpdateURLInput) error {
	u, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	if in.OriginalURL != "" {
		u.OriginalURL = in.OriginalURL
	}
	if in.Status != "" {
		switch in.Status {
		case model.StatusQueued, model.StatusRunning,
			model.StatusDone, model.StatusError, model.StatusStopped:
			u.Status = in.Status
		default:
			return errors.New("invalid status value")
		}
	}
	return s.repo.Update(u)
}

func NewURLService(r repository.URLRepository, p crawler.Pool) URLService {
	return &urlService{repo: r, crawlers: p}
}

func (s *urlService) Start(id uint) error {

	_, err := s.repo.FindByID(id)
	if err != nil {
		return fmt.Errorf("cannot start crawling: %w", err)
	}

	if err := s.repo.UpdateStatus(id, model.StatusQueued); err != nil {
		return err
	}
	s.crawlers.Enqueue(id)
	return nil
}

func (s *urlService) Stop(id uint) error {

	_, err := s.repo.FindByID(id)
	if err != nil {
		return fmt.Errorf("cannot stop crawling: %w", err)
	}

	return s.repo.UpdateStatus(id, model.StatusError)
}

func (s *urlService) Results(id uint) (*model.URLDTO, error) {
	url, err := s.repo.Results(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get URL results: %w", err)
	}
	return url.ToDTO(), nil
}

func (s *urlService) ResultsWithDetails(id uint) (*model.URL, []*model.AnalysisResult, []*model.Link, error) {
	url, analysisResults, links, err := s.repo.ResultsWithDetails(id)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get detailed URL results: %w", err)
	}

	return url, analysisResults, links, nil
}

func (s *urlService) Create(input *model.CreateURLInputDTO) (uint, error) {
	u := model.URLFromCreateInput(input)
	if err := s.repo.Create(u); err != nil {
		return 0, err
	}
	return u.ID, nil
}

func (s *urlService) Get(id uint) (*model.URLDTO, error) {
	u, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return u.ToDTO(), nil
}
func mapURLToDTO(url *model.URL) *model.URLDTO {
	return url.ToDTO()
}

func (s *urlService) List(userID uint, p repository.Pagination) (*model.PaginatedResponse[model.URLDTO], error) {
	urls, err := s.repo.ListByUser(userID, p)
	if err != nil {
		return nil, err
	}

	totalCount, err := s.repo.CountByUser(userID)
	if err != nil {
		return nil, err
	}

	totalPages := totalCount / p.PageSize
	if totalCount%p.PageSize > 0 {
		totalPages++
	}

	dtos := make([]model.URLDTO, len(urls))
	for i, url := range urls {
		dtos[i] = *mapURLToDTO(&url)
	}

	return &model.PaginatedResponse[model.URLDTO]{
		Data: dtos,
		Pagination: model.PaginationMetaDTO{
			Page:       p.Page,
			PageSize:   p.PageSize,
			TotalItems: totalCount,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *urlService) Delete(id uint) error {
	return s.repo.Delete(id)
}

func (s *urlService) StartWithPriority(id uint, priority int) error {

	_, err := s.repo.FindByID(id)
	if err != nil {
		return fmt.Errorf("cannot start crawling: %w", err)
	}

	if err := s.repo.UpdateStatus(id, model.StatusQueued); err != nil {
		return err
	}
	s.crawlers.EnqueueWithPriority(id, priority)
	return nil
}

func (s *urlService) GetCrawlResults() <-chan crawler.CrawlResult {
	return s.crawlers.GetResults()
}

func (s *urlService) AdjustCrawlerWorkers(action string, count int) error {
	if count <= 0 {
		return fmt.Errorf("worker count must be positive")
	}

	if action != "add" && action != "remove" {
		return fmt.Errorf("action must be 'add' or 'remove'")
	}

	s.crawlers.AdjustWorkers(crawler.ControlCommand{
		Action: action,
		Count:  count,
	})

	return nil
}
