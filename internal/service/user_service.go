package service

import (
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

type UserService interface {
	Register(input *model.CreateUserInput) (*model.UserDTO, error)
	Update(id uint, input *model.UpdateUserInput) (*model.UserDTO, error)
	Authenticate(email, password string) (*model.UserDTO, error)
	Get(id uint) (*model.UserDTO, error)
	Search(searchTerm, searchField, sortDirection string, p repository.Pagination) ([]*model.UserDTO, error)
	Delete(id uint) error
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) Register(input *model.CreateUserInput) (*model.UserDTO, error) {

	if existing, _ := s.repo.FindByEmail(input.Email); existing != nil {
		return nil, errors.New("email already in use")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &model.User{
		Username: input.Username,
		Email:    input.Email,
		Password: string(hash),
	}
	if err := s.repo.Create(u); err != nil {
		return nil, err
	}
	dto := u.ToDTO()
	return dto, nil
}

func (s *userService) Update(id uint, input *model.UpdateUserInput) (*model.UserDTO, error) {
	u, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if input.Username != nil {
		u.Username = *input.Username
	}
	if input.Email != nil {
		u.Email = *input.Email
	}
	if input.Password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		u.Password = string(hash)
	}
	if input.Role != nil {
		u.Role = *input.Role
	}
	if err := s.repo.Update(id, u); err != nil {
		return nil, err
	}
	return u.ToDTO(), nil
}

func (s *userService) Authenticate(email, password string) (*model.UserDTO, error) {
	u, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		return nil, errors.New("invalid credentials")
	}
	return u.ToDTO(), nil
}

func (s *userService) Get(id uint) (*model.UserDTO, error) {
	u, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return u.ToDTO(), nil
}

func (s *userService) Search(searchTerm, searchField, sortDirection string, p repository.Pagination) ([]*model.UserDTO, error) {
	users, err := s.repo.Search(searchTerm, searchField, sortDirection, p)
	if err != nil {
		return nil, err
	}
	dtos := make([]*model.UserDTO, len(users))
	for i, u := range users {
		dtos[i] = u.ToDTO()
	}
	return dtos, nil
}

func (s *userService) Delete(id uint) error {
	return s.repo.Delete(id)
}
