package middleware_test

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/middleware"
	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
)

// MockUserService is a mock implementation of the service.UserService interface.
type MockUserService struct {
	mock.Mock
}

// Authenticate mocks the user authentication.
func (m *MockUserService) Authenticate(email, password string) (*model.UserDTO, error) {
	args := m.Called(email, password)
	if result := args.Get(0); result != nil {
		return result.(*model.UserDTO), args.Error(1)
	}
	return nil, args.Error(1)
}

// Delete mocks the deletion of a user.
func (m *MockUserService) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// Get mocks fetching a user by ID.
func (m *MockUserService) Get(id uint) (*model.UserDTO, error) {
	args := m.Called(id)
	if result := args.Get(0); result != nil {
		return result.(*model.UserDTO), args.Error(1)
	}
	return nil, args.Error(1)
}

// List mocks listing users with pagination.
func (m *MockUserService) List(pagination repository.Pagination) ([]*model.UserDTO, error) {
	args := m.Called(pagination)
	if result := args.Get(0); result != nil {
		return result.([]*model.UserDTO), args.Error(1)
	}
	return nil, args.Error(1)
}

// Register mocks registering a new user. Note the interface expects a pointer to model.CreateUserInput.
func (m *MockUserService) Register(input *model.CreateUserInput) (*model.UserDTO, error) {
	args := m.Called(input)
	if result := args.Get(0); result != nil {
		return result.(*model.UserDTO), args.Error(1)
	}
	return nil, args.Error(1)
}

// MockTokenService is a mock implementation of the service.TokenService interface.
type MockTokenService struct {
	mock.Mock
}

// Validate mocks token validation and returns JWT claims.
func (m *MockTokenService) Validate(token string) (*service.JWTClaims, error) {
	args := m.Called(token)
	if result := args.Get(0); result != nil {
		return result.(*service.JWTClaims), args.Error(1)
	}
	return nil, args.Error(1)
}

// IsBlacklisted mocks checking if a token's ID (jti) is blacklisted.
func (m *MockTokenService) IsBlacklisted(tokenID string) (bool, error) {
	args := m.Called(tokenID)
	return args.Bool(0), args.Error(1)
}

// Generate mocks generating a JWT token for a given user.
func (m *MockTokenService) Generate(userID uint) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

// Invalidate mocks invalidating a JWT token by its ID.
func (m *MockTokenService) Invalidate(tokenID string) error {
	args := m.Called(tokenID)
	return args.Error(0)
}

// CleanupExpired mocks the removal of expired tokens.
func (m *MockTokenService) CleanupExpired() error {
	args := m.Called()
	return args.Error(0)
}

// MockUserLookup is a mock implementation of the service.UserLookup interface.
type MockUserLookup struct {
	mock.Mock
}

// FindByID mocks looking up a user by ID.
func (m *MockUserLookup) FindByID(id uint) (*model.User, error) {
	args := m.Called(id)
	if result := args.Get(0); result != nil {
		return result.(*model.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestBasicAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Define test cases for Basic HTTP Authentication
	tests := []struct {
		name           string
		headerValue    string
		setupMock      func(*MockUserService)
		expectedStatus int
	}{
		{
			name:           "Missing auth header",
			headerValue:    "",
			setupMock:      func(m *MockUserService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid prefix",
			headerValue:    "Bearer foo",
			setupMock:      func(m *MockUserService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Base64",
			headerValue:    "Basic invalidbase64",
			setupMock:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid basic auth format",
			headerValue:    "Basic " + base64.StdEncoding.EncodeToString([]byte("foo")),
			setupMock:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Authentication failure",
			headerValue: "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:wrongpassword")),
			setupMock: func(m *MockUserService) {
				// Expect Authenticate to be called and return an error.
				m.On("Authenticate", "user@example.com", "wrongpassword").Return(nil, errors.New("invalid credentials"))
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "Successful auth",
			headerValue: "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:correctpassword")),
			setupMock: func(m *MockUserService) {
				// Expect Authenticate and return a valid UserDTO.
				m.On("Authenticate", "user@example.com", "correctpassword").Return(&model.UserDTO{ID: 42}, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	// Iterate over test cases.
	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			mockUserService := new(MockUserService)
			tc.setupMock(mockUserService)

			// Setup Gin with the BasicAuthMiddleware.
			router := gin.New()
			router.Use(middleware.BasicAuthMiddleware(mockUserService))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "passed")
			})

			// Create the HTTP request.
			req, err := http.NewRequest("GET", "/test", nil)
			require.NoError(t, err)
			if tc.headerValue != "" {
				req.Header.Set("Authorization", tc.headerValue)
			}

			// Record the response.
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				require.Equal(t, "passed", w.Body.String())
			}

			// Verify all expected calls on the mock.
			mockUserService.AssertExpectations(t)
		})
	}
}

func TestJWTAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Define test cases for JWT Authentication
	tests := []struct {
		name           string
		headerValue    string
		setupTokenMock func(*MockTokenService)
		setupUserMock  func(*MockUserLookup)
		expectedStatus int
	}{
		{
			name:           "Missing auth header",
			headerValue:    "",
			setupTokenMock: func(m *MockTokenService) {},
			setupUserMock:  func(m *MockUserLookup) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid prefix",
			headerValue:    "Basic foo",
			setupTokenMock: func(m *MockTokenService) {},
			setupUserMock:  func(m *MockUserLookup) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "Token validation fails",
			headerValue: "Bearer invalidtoken",
			setupTokenMock: func(m *MockTokenService) {
				// Expect Validate to fail
				m.On("Validate", "invalidtoken").Return(nil, errors.New("invalid token"))
			},
			setupUserMock:  func(m *MockUserLookup) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "Token blacklisted",
			headerValue: "Bearer validtoken",
			setupTokenMock: func(m *MockTokenService) {
				// Create claims with a valid ID using RegisteredClaims.ID field.
				claims := &service.JWTClaims{
					RegisteredClaims: jwt.RegisteredClaims{
						ID:        "abc123",
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
					},
					UserID: 42,
				}
				m.On("Validate", "validtoken").Return(claims, nil)
				// Expect the token to be blacklisted.
				m.On("IsBlacklisted", "abc123").Return(true, nil)
			},
			setupUserMock:  func(m *MockUserLookup) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "User no longer exists",
			headerValue: "Bearer validtoken",
			setupTokenMock: func(m *MockTokenService) {
				claims := &service.JWTClaims{
					RegisteredClaims: jwt.RegisteredClaims{
						ID:        "abc123",
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
					},
					UserID: 42,
				}
				m.On("Validate", "validtoken").Return(claims, nil)
				m.On("IsBlacklisted", "abc123").Return(false, nil)
			},
			setupUserMock: func(m *MockUserLookup) {
				// Expect FindByID to fail if user is not found.
				m.On("FindByID", uint(42)).Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "Successful JWT auth",
			headerValue: "Bearer validtoken",
			setupTokenMock: func(m *MockTokenService) {
				claims := &service.JWTClaims{
					RegisteredClaims: jwt.RegisteredClaims{
						ID:        "abc123",
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
					},
					UserID: 42,
				}
				m.On("Validate", "validtoken").Return(claims, nil)
				m.On("IsBlacklisted", "abc123").Return(false, nil)
			},
			setupUserMock: func(m *MockUserLookup) {
				// Expect to find a valid user.
				m.On("FindByID", uint(42)).Return(&model.User{ID: 42}, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	// Iterate over JWT auth test cases.
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockTokenService := new(MockTokenService)
			mockUserLookup := new(MockUserLookup)

			tc.setupTokenMock(mockTokenService)
			tc.setupUserMock(mockUserLookup)

			// Setup Gin with the JWTAuthMiddleware.
			router := gin.New()
			router.Use(middleware.JWTAuthMiddleware(mockTokenService, mockUserLookup))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "jwt passed")
			})

			// Create the HTTP request.
			req, err := http.NewRequest("GET", "/test", nil)
			require.NoError(t, err)
			if tc.headerValue != "" {
				req.Header.Set("Authorization", tc.headerValue)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, tc.expectedStatus, w.Code)
			if tc.expectedStatus == http.StatusOK {
				require.Equal(t, "jwt passed", w.Body.String())
			}

			// Verify all expected calls on the mocks.
			mockTokenService.AssertExpectations(t)
			mockUserLookup.AssertExpectations(t)
		})
	}
}
