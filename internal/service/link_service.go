package service

import (
	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// LinkService handles link persistence.
type LinkService interface {
	Add(link *model.Link) error
	List(urlID uint, p repository.Pagination) ([]*model.LinkDTO, error)
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
