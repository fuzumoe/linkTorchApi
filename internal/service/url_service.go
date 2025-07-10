package service

import (
	"errors"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// URLService defines business operations around URLs.
type URLService interface {
	Create(input *model.CreateURLInput) (uint, error)
	Get(id uint) (*model.URLDTO, error)
	List(userID uint, p repository.Pagination) ([]*model.URLDTO, error)
	Update(id uint, input *model.UpdateURLInput) error
	Delete(id uint) error
}

type urlService struct {
	repo repository.URLRepository
}

// NewURLService constructs a URLService.
func NewURLService(repo repository.URLRepository) URLService {
	return &urlService{repo: repo}
}

func (s *urlService) Create(input *model.CreateURLInput) (uint, error) {
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

func (s *urlService) List(userID uint, p repository.Pagination) ([]*model.URLDTO, error) {
	urls, err := s.repo.ListByUser(userID, p)
	if err != nil {
		return nil, err
	}
	dtos := make([]*model.URLDTO, len(urls))
	for i, u := range urls {
		dtos[i] = u.ToDTO()
	}
	return dtos, nil
}

func (s *urlService) Update(id uint, input *model.UpdateURLInput) error {
	u, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if input.OriginalURL != "" {
		u.OriginalURL = input.OriginalURL
	}
	if input.Status != "" {
		switch input.Status {
		case model.StatusQueued, model.StatusRunning, model.StatusDone, model.StatusError:
			u.Status = input.Status
		default:
			return errors.New("invalid status value")
		}
	}
	return s.repo.Update(u)
}

func (s *urlService) Delete(id uint) error {
	return s.repo.Delete(id)
}
