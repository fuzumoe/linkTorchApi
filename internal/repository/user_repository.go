package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

// UserRepository defines all DB operations around users.
type UserRepository interface {
	Create(u *model.User) error
	FindByID(id uint) (*model.User, error)
	FindByEmail(email string) (*model.User, error)
	ListAll() ([]model.User, error)
	Delete(id uint) error
}

// userRepo is the GORM implementation of UserRepository.
type userRepo struct {
	db *gorm.DB
}

// NewUserRepo returns a UserRepository backed by GORM.
func NewUserRepo(db *gorm.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(u *model.User) error {
	return r.db.Create(u).Error
}

func (r *userRepo) FindByID(id uint) (*model.User, error) {
	var u model.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) FindByEmail(email string) (*model.User, error) {
	var u model.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) ListAll() ([]model.User, error) {
	var users []model.User
	if err := r.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *userRepo) Delete(id uint) error {
	res := r.db.Delete(&model.User{}, id)
	if res.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return res.Error
}
