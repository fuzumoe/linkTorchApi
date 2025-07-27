package service

import (
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

// LinkService handles link persistence.
type LinkService interface {
	Add(link *model.Link) error
	List(urlID uint, p repository.Pagination) ([]*model.LinkDTO, error)
	ListByURL(urlID uint, p repository.Pagination) (*model.PaginatedResponse[model.LinkDTO], error)
	Update(link *model.Link) error
	Delete(link *model.Link) error
}

type linkService struct {
	repo repository.LinkRepository
}

// NewLinkService constructs a LinkService.
func NewLinkService(repo repository.LinkRepository) LinkService {
	return &linkService{repo: repo}
}

func (s *linkService) ListByURL(urlID uint, p repository.Pagination) (*model.PaginatedResponse[model.LinkDTO], error) {
	links, err := s.repo.ListByURL(urlID, p)
	if err != nil {
		return nil, err
	}

	// Get total count for pagination metadata
	totalCount, err := s.repo.CountByURL(urlID)
	if err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := totalCount / p.PageSize
	if totalCount%p.PageSize > 0 {
		totalPages++
	}

	// Convert models to DTOs
	dtos := make([]model.LinkDTO, len(links))
	for i, link := range links {
		dtos[i] = mapLinkToDTO(&link)
	}

	return &model.PaginatedResponse[model.LinkDTO]{
		Data: dtos,
		Pagination: model.PaginationMetaDTO{
			Page:       p.Page,
			PageSize:   p.PageSize,
			TotalItems: totalCount,
			TotalPages: totalPages,
		},
	}, nil
}

func mapLinkToDTO(link *model.Link) model.LinkDTO {
	return model.LinkDTO{
		ID:         link.ID,
		URLID:      link.URLID,
		Href:       link.Href,
		IsExternal: link.IsExternal,
		StatusCode: link.StatusCode,
		CreatedAt:  link.CreatedAt,
		UpdatedAt:  link.UpdatedAt,
	}
}
func (s *linkService) Add(link *model.Link) error {
	return s.repo.Create(link)
}

func (s *linkService) List(urlID uint, p repository.Pagination) ([]*model.LinkDTO, error) {
	links, err := s.repo.ListByURL(urlID, p)
	if err != nil {
		return nil, err
	}
	dtos := make([]*model.LinkDTO, len(links))
	for i, l := range links {
		dtos[i] = l.ToDTO()
	}
	return dtos, nil
}

func (s *linkService) Update(link *model.Link) error {
	return s.repo.Update(link)
}

func (s *linkService) Delete(link *model.Link) error {
	return s.repo.Delete(link)
}
