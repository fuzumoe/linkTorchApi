package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/handler"
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Register(input *model.CreateUserInput) (*model.UserDTO, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserDTO), args.Error(1)
}

func (m *MockUserService) Get(id uint) (*model.UserDTO, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserDTO), args.Error(1)
}

func (m *MockUserService) Update(id uint, input *model.UpdateUserInput) (*model.UserDTO, error) {
	args := m.Called(id, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserDTO), args.Error(1)
}

func (m *MockUserService) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserService) Authenticate(email, password string) (*model.UserDTO, error) {
	args := m.Called(email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserDTO), args.Error(1)
}

func (m *MockUserService) Search(query, sort, filter string, p repository.Pagination) ([]*model.UserDTO, error) {
	args := m.Called(query, sort, filter, p)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.UserDTO), args.Error(1)
}

func setupUserHandler(_ *testing.T, userRole string) (*gin.Engine, *MockUserService) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("user_role", userRole)
		c.Next()
	})

	userService := &MockUserService{}
	userHandler := handler.NewUserHandler(userService)

	apiGroup := r.Group("/api")
	userHandler.RegisterProtectedRoutes(apiGroup)

	return r, userService
}

func TestUserCreate(t *testing.T) {
	r, userService := setupUserHandler(t, "admin")

	newUser := &model.UserDTO{
		ID:       42,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     model.RoleUser,
	}

	userService.On("Register", &model.CreateUserInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}).Return(newUser, nil)

	reqBody := []byte(`{
        "username": "testuser",
        "email": "test@example.com",
        "password": "password123"
    }`)
	req, _ := http.NewRequest(http.MethodPost, "/api/users", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	userData, ok := response["id"].(map[string]interface{})
	require.True(t, ok, "Response should contain user data in 'id' field")
	assert.Equal(t, float64(42), userData["id"])
	assert.Equal(t, "testuser", userData["username"])
	assert.Equal(t, "test@example.com", userData["email"])

	userService.AssertExpectations(t)
}

func TestUserCreateError(t *testing.T) {
	r, userService := setupUserHandler(t, "admin")

	userService.On("Register", mock.Anything).Return(nil, errors.New("creation failed"))

	reqBody := []byte(`{
        "username": "testuser",
        "email": "test@example.com",
        "password": "password123"
    }`)
	req, _ := http.NewRequest(http.MethodPost, "/api/users", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	userService.AssertExpectations(t)
}

func TestUserMe(t *testing.T) {
	r, userService := setupUserHandler(t, "user")

	userService.On("Get", uint(1)).Return(&model.UserDTO{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     model.RoleUser,
	}, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/users/me", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(1), response["id"])
	assert.Equal(t, "testuser", response["username"])
	assert.Equal(t, "test@example.com", response["email"])
	assert.Equal(t, "user", response["role"])

	userService.AssertExpectations(t)
}

func TestUserSearch(t *testing.T) {
	r, userService := setupUserHandler(t, "admin")

	userService.On("Search", "test", "", "", repository.Pagination{
		Page:     1,
		PageSize: 10,
	}).Return([]*model.UserDTO{
		{
			ID:       1,
			Username: "testuser1",
			Email:    "test1@example.com",
			Role:     model.RoleUser,
		},
		{
			ID:       2,
			Username: "testuser2",
			Email:    "test2@example.com",
			Role:     model.RoleAdmin,
		},
	}, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/users/search?q=test&page=1&page_size=10", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response, 2)
	assert.Equal(t, "testuser1", response[0]["username"])
	assert.Equal(t, "testuser2", response[1]["username"])

	userService.AssertExpectations(t)
}

func TestUserSearchNoAdminRole(t *testing.T) {
	r, _ := setupUserHandler(t, "user")

	req, _ := http.NewRequest(http.MethodGet, "/api/users/search?q=test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUserGetById(t *testing.T) {
	r, userService := setupUserHandler(t, "admin")

	userService.On("Search", "anything", "", "", mock.Anything).Return([]*model.UserDTO{
		{
			ID:       42,
			Username: "testuser",
			Email:    "test@example.com",
			Role:     model.RoleUser,
		},
	}, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/users/42?q=anything", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response, 1)
	assert.Equal(t, float64(42), response[0]["id"])
	assert.Equal(t, "testuser", response[0]["username"])

	userService.AssertExpectations(t)
}

func TestUserUpdate(t *testing.T) {
	r, userService := setupUserHandler(t, "user")

	updatedUser := &model.UserDTO{
		ID:       1,
		Username: "updateduser",
		Email:    "updated@example.com",
		Role:     model.RoleUser,
	}

	username := "updateduser"
	email := "updated@example.com"

	userService.On("Update", uint(1), &model.UpdateUserInput{
		Username: &username,
		Email:    &email,
	}).Return(updatedUser, nil)

	reqBody := []byte(`{
        "username": "updateduser",
        "email": "updated@example.com"
    }`)
	req, _ := http.NewRequest(http.MethodPut, "/api/users/1", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "updateduser", response["username"])
	assert.Equal(t, "updated@example.com", response["email"])
	assert.Equal(t, "user", response["role"])

	userService.AssertExpectations(t)
}

func TestUserUpdateOtherUserForbidden(t *testing.T) {

	r, _ := setupUserHandler(t, "user")

	reqBody := []byte(`{
        "username": "updateduser",
        "email": "updated@example.com"
    }`)
	req, _ := http.NewRequest(http.MethodPut, "/api/users/2", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUserUpdateRoleAsAdmin(t *testing.T) {

	r, userService := setupUserHandler(t, "admin")

	role := model.RoleAdmin
	updatedUser := &model.UserDTO{
		ID:       2,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     model.RoleAdmin,
	}

	userService.On("Update", uint(2), &model.UpdateUserInput{
		Role: &role,
	}).Return(updatedUser, nil)

	reqBody := []byte(`{
        "role": "admin"
    }`)
	req, _ := http.NewRequest(http.MethodPut, "/api/users/2", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	userService.AssertExpectations(t)
}

func TestUserDelete(t *testing.T) {
	r, userService := setupUserHandler(t, "admin")

	userService.On("Delete", uint(2)).Return(nil)

	req, _ := http.NewRequest(http.MethodDelete, "/api/users/2", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	userService.AssertExpectations(t)
}

func TestUserDeleteForbiddenForNonAdmin(t *testing.T) {

	r, _ := setupUserHandler(t, "user")

	req, _ := http.NewRequest(http.MethodDelete, "/api/users/2", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUserDeleteInvalidID(t *testing.T) {
	r, _ := setupUserHandler(t, "admin")

	req, _ := http.NewRequest(http.MethodDelete, "/api/users/invalid", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
