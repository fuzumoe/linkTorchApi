package handler_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/fuzumoe/linkTorch-api/internal/handler"
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/internal/service"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) AuthenticateBasic(email, password string) (*model.UserDTO, error) {
	args := m.Called(email, password)
	if user, ok := args.Get(0).(*model.UserDTO); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) Validate(token string) (*service.Claims, error) {
	args := m.Called(token)
	if claims, ok := args.Get(0).(*service.Claims); ok {
		return claims, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) Generate(userID uint) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) Invalidate(tokenID string) error {
	args := m.Called(tokenID)
	return args.Error(0)
}

func (m *MockAuthService) CleanupExpired() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockAuthService) IsTokenRevoked(tokenID string) (bool, error) {
	args := m.Called(tokenID)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthService) FindUserById(userID uint) (*model.UserDTO, error) {
	args := m.Called(userID)
	if user, ok := args.Get(0).(*model.UserDTO); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Authenticate(email, password string) (*model.UserDTO, error) {
	args := m.Called(email, password)
	if user, ok := args.Get(0).(*model.UserDTO); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) Register(input *model.CreateUserInput) (*model.UserDTO, error) {
	args := m.Called(input)
	if user, ok := args.Get(0).(*model.UserDTO); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) Update(id uint, input *model.UpdateUserInput) (*model.UserDTO, error) {
	args := m.Called(id, input)
	if user, ok := args.Get(0).(*model.UserDTO); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) Delete(userID uint) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserService) Get(userID uint) (*model.UserDTO, error) {
	args := m.Called(userID)
	if user, ok := args.Get(0).(*model.UserDTO); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) Search(email string, role string, username string, p repository.Pagination) ([]*model.UserDTO, error) {
	args := m.Called(email, role, username, p)
	if users, ok := args.Get(0).([]*model.UserDTO); ok {
		return users, args.Error(1)
	}
	return nil, args.Error(1)
}

func TestLoginBasic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authService := new(MockAuthService)
	userService := new(MockUserService)
	h := handler.NewAuthHandler(authService, userService)

	testEmail := "test@example.com"
	testPassword := "password123"
	userDTO := &model.UserDTO{
		ID:    1,
		Email: testEmail,
	}

	userService.On("Authenticate", testEmail, testPassword).Return(userDTO, nil)
	authService.On("Generate", uint(1)).Return("JWT-TOKEN", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	creds := testEmail + ":" + testPassword
	encoded := base64.StdEncoding.EncodeToString([]byte(creds))
	req, _ := http.NewRequest(http.MethodPost, "/login/basic", nil)
	req.Header.Set("Authorization", "Basic "+encoded)
	c.Request = req

	h.LoginBasic(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "JWT-TOKEN", resp["token"])

	userService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestLoginJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authService := new(MockAuthService)
	userService := new(MockUserService)
	h := handler.NewAuthHandler(authService, userService)

	testEmail := "test@example.com"
	testPassword := "password123"
	userDTO := &model.UserDTO{
		ID:    2,
		Email: testEmail,
	}

	userService.On("Authenticate", testEmail, testPassword).Return(userDTO, nil)
	authService.On("Generate", uint(2)).Return("JWT-TOKEN-JWT", nil)

	payload := map[string]string{
		"email":    testEmail,
		"password": testPassword,
	}
	payloadBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/login/jwt", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	h.LoginJWT(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "JWT-TOKEN-JWT", resp["token"])
	userService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authService := new(MockAuthService)
	userService := new(MockUserService)
	h := handler.NewAuthHandler(authService, userService)

	tokenStr := "TestBearerToken"
	claims := &service.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID: "unique-token-id",
		},
	}
	authService.On("Validate", tokenStr).Return(claims, nil)
	authService.On("Invalidate", "unique-token-id").Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	h.Logout(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "logged out", resp["message"])
	authService.AssertExpectations(t)
}
