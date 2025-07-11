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
	"github.com/fuzumoe/urlinsight-backend/internal/service"
)

// MockAuthService implements service.AuthService for testing.
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) AuthenticateBasic(email, password string) (*model.UserDTO, error) {
	args := m.Called(email, password)
	if res := args.Get(0); res != nil {
		return res.(*model.UserDTO), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) Validate(token string) (*service.Claims, error) {
	args := m.Called(token)
	if res := args.Get(0); res != nil {
		return res.(*service.Claims), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) IsTokenRevoked(tokenID string) (bool, error) {
	args := m.Called(tokenID)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthService) FindUserById(userID uint) (*model.UserDTO, error) {
	args := m.Called(userID)
	if res := args.Get(0); res != nil {
		return res.(*model.UserDTO), args.Error(1)
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

// Testing the single AuthMiddleware function that handles both Basic and JWT auth
func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Missing Auth Header", func(t *testing.T) {
		mockAuth := new(MockAuthService)

		router := gin.New()
		router.Use(middleware.AuthMiddleware(mockAuth))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "passed")
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Basic Auth Flow", func(t *testing.T) {
		tests := []struct {
			name           string
			headerValue    string
			setupMock      func(m *MockAuthService)
			expectedStatus int
		}{
			{
				name:           "Invalid Base64",
				headerValue:    "Basic invalidbase64",
				setupMock:      func(m *MockAuthService) {},
				expectedStatus: http.StatusBadRequest,
			},
			{
				name:           "Invalid basic auth format",
				headerValue:    "Basic " + base64.StdEncoding.EncodeToString([]byte("foo")),
				setupMock:      func(m *MockAuthService) {},
				expectedStatus: http.StatusBadRequest,
			},
			{
				name:        "Authentication failure",
				headerValue: "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:wrongpassword")),
				setupMock: func(m *MockAuthService) {
					m.On("AuthenticateBasic", "user@example.com", "wrongpassword").
						Return(nil, errors.New("invalid credentials"))
				},
				expectedStatus: http.StatusUnauthorized,
			},
			{
				name:        "Successful auth",
				headerValue: "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:correctpassword")),
				setupMock: func(m *MockAuthService) {
					m.On("AuthenticateBasic", "user@example.com", "correctpassword").
						Return(&model.UserDTO{ID: 42, Username: "testuser", Email: "user@example.com"}, nil)
				},
				expectedStatus: http.StatusOK,
			},
		}

		for _, tc := range tests {
			tc := tc // capture range var
			t.Run(tc.name, func(t *testing.T) {
				mockAuth := new(MockAuthService)
				tc.setupMock(mockAuth)

				router := gin.New()
				router.Use(middleware.AuthMiddleware(mockAuth))
				router.GET("/test", func(c *gin.Context) {
					c.String(http.StatusOK, "passed")
				})

				req, err := http.NewRequest("GET", "/test", nil)
				require.NoError(t, err)
				req.Header.Set("Authorization", tc.headerValue)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				require.Equal(t, tc.expectedStatus, w.Code)
				if tc.expectedStatus == http.StatusOK {
					require.Equal(t, "passed", w.Body.String())
				}

				mockAuth.AssertExpectations(t)
			})
		}
	})

	t.Run("JWT Auth Flow", func(t *testing.T) {
		tests := []struct {
			name           string
			headerValue    string
			setupMock      func(m *MockAuthService)
			expectedStatus int
		}{
			{
				name:        "Invalid prefix",
				headerValue: "Bearer foo",
				setupMock: func(m *MockAuthService) {
					// Even with invalid format, middleware still tries to validate
					m.On("Validate", "foo").Return(nil, errors.New("invalid token"))
				},
				expectedStatus: http.StatusUnauthorized,
			},
			{
				name:           "Unsupported auth type",
				headerValue:    "Digest something",
				setupMock:      func(m *MockAuthService) {},
				expectedStatus: http.StatusUnauthorized,
			},
			{
				name:        "Token validation fails",
				headerValue: "Bearer invalidtoken",
				setupMock: func(m *MockAuthService) {
					m.On("Validate", "invalidtoken").Return(nil, errors.New("invalid token"))
				},
				expectedStatus: http.StatusUnauthorized,
			},
			{
				name:        "Token blacklisted",
				headerValue: "Bearer validtoken",
				setupMock: func(m *MockAuthService) {
					claims := &service.Claims{
						RegisteredClaims: jwt.RegisteredClaims{
							ID:        "abc123",
							ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
						},
						UserID: 42,
					}
					m.On("Validate", "validtoken").Return(claims, nil)
					m.On("IsTokenRevoked", "abc123").Return(true, nil)
				},
				expectedStatus: http.StatusUnauthorized,
			},
			{
				name:        "User no longer exists",
				headerValue: "Bearer validtoken",
				setupMock: func(m *MockAuthService) {
					claims := &service.Claims{
						RegisteredClaims: jwt.RegisteredClaims{
							ID:        "abc123",
							ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
						},
						UserID: 42,
					}
					m.On("Validate", "validtoken").Return(claims, nil)
					m.On("IsTokenRevoked", "abc123").Return(false, nil)
					m.On("FindUserById", uint(42)).Return(nil, errors.New("user not found"))
				},
				expectedStatus: http.StatusUnauthorized,
			},
			{
				name:        "Successful JWT auth",
				headerValue: "Bearer validtoken",
				setupMock: func(m *MockAuthService) {
					claims := &service.Claims{
						RegisteredClaims: jwt.RegisteredClaims{
							ID:        "abc123",
							ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
						},
						UserID: 42,
					}
					m.On("Validate", "validtoken").Return(claims, nil)
					m.On("IsTokenRevoked", "abc123").Return(false, nil)
					m.On("FindUserById", uint(42)).Return(&model.UserDTO{ID: 42, Username: "testuser", Email: "user@example.com"}, nil)
				},
				expectedStatus: http.StatusOK,
			},
		}

		for _, tc := range tests {
			tc := tc // capture range var
			t.Run(tc.name, func(t *testing.T) {
				mockAuth := new(MockAuthService)
				tc.setupMock(mockAuth)

				router := gin.New()
				router.Use(middleware.AuthMiddleware(mockAuth))
				router.GET("/test", func(c *gin.Context) {
					c.String(http.StatusOK, "jwt passed")
				})

				req, err := http.NewRequest("GET", "/test", nil)
				require.NoError(t, err)
				req.Header.Set("Authorization", tc.headerValue)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				require.Equal(t, tc.expectedStatus, w.Code)
				if tc.expectedStatus == http.StatusOK {
					require.Equal(t, "jwt passed", w.Body.String())
				}

				mockAuth.AssertExpectations(t)
			})
		}
	})
}
