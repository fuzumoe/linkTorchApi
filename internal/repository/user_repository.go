package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

// UserRepository defines all DB operations around users.
type UserRepository interface {
	Create(u *model.User) error
	Update(id uint, u *model.User) error
	FindByID(id uint) (*model.User, error)
	FindByEmail(email string) (*model.User, error)
	Search(email, role, username string, p Pagination) ([]model.User, error)
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

func (r *userRepo) Update(id uint, u *model.User) error {
	return r.db.Model(&model.User{ID: id}).Updates(u).Error
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

func (r *userRepo) Search(email, role, username string, p Pagination) ([]model.User, error) {
	var users []model.User
	query := r.db
	if email != "" {
		query = query.Where("email LIKE ?", "%"+email+"%")
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}

	err := query.
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&users).Error

	return users, err
}

func (r *userRepo) Delete(id uint) error {
	res := r.db.Delete(&model.User{}, id)
	if res.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return res.Error
}
