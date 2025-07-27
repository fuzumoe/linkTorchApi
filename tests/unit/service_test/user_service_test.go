package service_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/internal/service"
)

type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) Search(email string, role string, username string, p repository.Pagination) ([]model.User, error) {
	args := m.Called(email, role, username, p)
	return args.Get(0).([]model.User), args.Error(1)
}

func (m *MockUserRepo) Update(id uint, u *model.User) error {
	args := m.Called(id, u)
	return args.Error(0)
}

func (m *MockUserRepo) Create(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepo) FindByID(id uint) (*model.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepo) FindByEmail(email string) (*model.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepo) ListAll(p repository.Pagination) ([]model.User, error) {
	args := m.Called(p)
	return args.Get(0).([]model.User), args.Error(1)
}

func (m *MockUserRepo) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestUserService_Register(t *testing.T) {

	mockRepo := new(MockUserRepo)
	svc := service.NewUserService(mockRepo)

	input := &model.CreateUserInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("FindByEmail", input.Email).Return(nil, errors.New("not found")).Once()
		mockRepo.On("Create", mock.AnythingOfType("*model.User")).Run(func(args mock.Arguments) {
			user := args.Get(0).(*model.User)
			assert.NotEqual(t, input.Password, user.Password, "Password should be hashed")

			user.ID = 1
		}).Return(nil).Once()

		dto, err := svc.Register(input)

		require.NoError(t, err)
		assert.NotNil(t, dto)
		assert.Equal(t, uint(1), dto.ID)
		assert.Equal(t, input.Username, dto.Username)
		assert.Equal(t, input.Email, dto.Email)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Email Already Exists", func(t *testing.T) {

		existingUser := &model.User{
			ID:       1,
			Username: "existing",
			Email:    input.Email,
			Password: "hashedpw",
		}
		mockRepo.On("FindByEmail", input.Email).Return(existingUser, nil).Once()

		dto, err := svc.Register(input)

		assert.Error(t, err)
		assert.Equal(t, "email already in use", err.Error())
		assert.Nil(t, dto)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {

		mockRepo.On("FindByEmail", input.Email).Return(nil, errors.New("not found")).Once()
		mockRepo.On("Create", mock.AnythingOfType("*model.User")).Return(errors.New("db error")).Once()

		dto, err := svc.Register(input)

		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		assert.Nil(t, dto)
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_Authenticate(t *testing.T) {

	mockRepo := new(MockUserRepo)
	svc := service.NewUserService(mockRepo)

	email := "test@example.com"
	password := "password123"

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	user := &model.User{
		ID:       1,
		Username: "testuser",
		Email:    email,
		Password: string(hashedPassword),
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("FindByEmail", email).Return(user, nil).Once()

		dto, err := svc.Authenticate(email, password)

		require.NoError(t, err)
		assert.NotNil(t, dto)
		assert.Equal(t, user.ID, dto.ID)
		assert.Equal(t, user.Username, dto.Username)
		assert.Equal(t, user.Email, dto.Email)
		mockRepo.AssertExpectations(t)
	})

	t.Run("User Not Found", func(t *testing.T) {
		mockRepo.On("FindByEmail", email).Return(nil, errors.New("not found")).Once()

		dto, err := svc.Authenticate(email, password)

		assert.Error(t, err)
		assert.Equal(t, "invalid credentials", err.Error())
		assert.Nil(t, dto)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Wrong Password", func(t *testing.T) {
		mockRepo.On("FindByEmail", email).Return(user, nil).Once()

		dto, err := svc.Authenticate(email, "wrongpassword")
		assert.Error(t, err)
		assert.Equal(t, "invalid credentials", err.Error())
		assert.Nil(t, dto)
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_Get(t *testing.T) {

	mockRepo := new(MockUserRepo)
	svc := service.NewUserService(mockRepo)

	userID := uint(1)
	user := &model.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("FindByID", userID).Return(user, nil).Once()

		dto, err := svc.Get(userID)
		require.NoError(t, err)
		assert.NotNil(t, dto)
		assert.Equal(t, user.ID, dto.ID)
		assert.Equal(t, user.Username, dto.Username)
		assert.Equal(t, user.Email, dto.Email)

		mockRepo.AssertExpectations(t)
	})

	t.Run("User Not Found", func(t *testing.T) {
		mockRepo.On("FindByID", userID).Return(nil, errors.New("not found")).Once()

		dto, err := svc.Get(userID)

		assert.Error(t, err)
		assert.Equal(t, "not found", err.Error())
		assert.Nil(t, dto)
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_List(t *testing.T) {
	mockRepo := new(MockUserRepo)
	svc := service.NewUserService(mockRepo)

	pagination := repository.Pagination{Page: 1, PageSize: 10}
	users := []model.User{
		{
			ID:       1,
			Username: "user1",
			Email:    "user1@example.com",
			Password: "hashedpw1",
		},
		{
			ID:       2,
			Username: "user2",
			Email:    "user2@example.com",
			Password: "hashedpw2",
		},
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Search", "", "", "", pagination).Return(users, nil).Once()

		dtos, err := svc.Search("", "", "", pagination)

		require.NoError(t, err)
		require.Len(t, dtos, 2)

		assert.Equal(t, users[0].ID, dtos[0].ID)
		assert.Equal(t, users[0].Username, dtos[0].Username)
		assert.Equal(t, users[0].Email, dtos[0].Email)
		assert.Equal(t, users[1].ID, dtos[1].ID)
		assert.Equal(t, users[1].Username, dtos[1].Username)
		assert.Equal(t, users[1].Email, dtos[1].Email)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Empty Search", func(t *testing.T) {
		mockRepo.On("Search", "", "", "", pagination).Return([]model.User{}, nil).Once()

		dtos, err := svc.Search("", "", "", pagination)

		require.NoError(t, err)
		assert.Empty(t, dtos)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {

		mockRepo.On("Search", "", "", "", pagination).Return([]model.User{}, errors.New("db error")).Once()

		dtos, err := svc.Search("", "", "", pagination)

		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		assert.Nil(t, dtos)
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_Delete(t *testing.T) {

	mockRepo := new(MockUserRepo)
	svc := service.NewUserService(mockRepo)

	userID := uint(1)

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Delete", userID).Return(nil).Once()

		err := svc.Delete(userID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("User Not Found", func(t *testing.T) {
		mockRepo.On("Delete", userID).Return(errors.New("user not found")).Once()

		err := svc.Delete(userID)

		assert.Error(t, err)
		assert.Equal(t, "user not found", err.Error())
		mockRepo.AssertExpectations(t)
	})
}
