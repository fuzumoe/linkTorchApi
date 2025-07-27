package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/internal/service"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(id uint) (*model.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(email string) (*model.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) Update(id uint, u *model.User) error {
	args := m.Called(id, u)
	return args.Error(0)
}

func (m *MockUserRepository) Search(email string, role string, username string, p repository.Pagination) ([]model.User, error) {
	args := m.Called(email, role, username, p)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.User), args.Error(1)
}

type MockTokenRepository struct {
	mock.Mock
}

func (m *MockTokenRepository) Add(token *model.BlacklistedToken) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockTokenRepository) IsBlacklisted(jti string) (bool, error) {
	args := m.Called(jti)
	return args.Bool(0), args.Error(1)
}

func (m *MockTokenRepository) RemoveExpired() error {
	args := m.Called()
	return args.Error(0)
}

func createTestUser(id uint) *model.User {
	validHash := "$2a$10$DwPN33P/gX.yrFZ7Vw4GpuScqXd2QrQJtBSmPnxLrhS/Pv7T/Kvja"

	return &model.User{
		ID:       id,
		Username: "testuser",
		Email:    "test@example.com",
		Password: validHash,
		Role:     "user",
	}
}

func TestAuthService_AuthenticateBasic(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtSecret, tokenLifetime)

	t.Run("Success", func(t *testing.T) {
		email := "test@example.com"
		password := "password123"
		user := createTestUser(1)

		mockUserRepo.On("FindByEmail", email).Return(user, nil).Once()

		svc.AuthenticateBasic(email, password)
		mockUserRepo.AssertCalled(t, "FindByEmail", email)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("User Not Found", func(t *testing.T) {
		email := "nonexistent@example.com"
		password := "password123"

		mockUserRepo.On("FindByEmail", email).Return(nil, errors.New("user not found")).Once()

		userDTO, err := svc.AuthenticateBasic(email, password)
		assert.Error(t, err)
		assert.Nil(t, userDTO)
		assert.Equal(t, "invalid credentials", err.Error())
		mockUserRepo.AssertExpectations(t)
	})
}

func TestAuthService_Generate(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtSecret, tokenLifetime)

	userID := uint(123)

	t.Run("Success", func(t *testing.T) {
		user := createTestUser(userID)

		mockUserRepo.On("FindByID", userID).Return(user, nil).Once()

		tokenString, err := svc.Generate(userID)
		require.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		token, err := jwt.ParseWithClaims(tokenString, &service.Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		require.NoError(t, err)

		claims, ok := token.Claims.(*service.Claims)
		require.True(t, ok)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Role, claims.Role)
		assert.NotEmpty(t, claims.ID)

		now := time.Now().UTC()
		assert.WithinDuration(t, now, claims.IssuedAt.Time, 2*time.Second)
		assert.WithinDuration(t, now.Add(tokenLifetime), claims.ExpiresAt.Time, 2*time.Second)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("User Not Found", func(t *testing.T) {
		mockUserRepo.On("FindByID", userID).Return(nil, errors.New("user not found")).Once()

		tokenString, err := svc.Generate(userID)
		assert.Error(t, err)
		assert.Empty(t, tokenString)
		assert.Equal(t, "user not found", err.Error())
		mockUserRepo.AssertExpectations(t)
	})
}

func TestAuthService_Validate(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtSecret, tokenLifetime)

	userID := uint(123)

	mockUserRepo.On("FindByID", userID).Return(createTestUser(userID), nil).Once()
	validToken, err := svc.Generate(userID)
	require.NoError(t, err)

	token, _ := jwt.ParseWithClaims(validToken, &service.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	claims := token.Claims.(*service.Claims)
	tokenID := claims.ID

	t.Run("Valid Token", func(t *testing.T) {
		mockTokenRepo.On("IsBlacklisted", tokenID).Return(false, nil).Once()

		claims, err := svc.Validate(validToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.NotEmpty(t, claims.ID)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Invalid Token Format", func(t *testing.T) {
		claims, err := svc.Validate("invalid-token-format")
		assert.Error(t, err)
		assert.Equal(t, service.ErrTokenInvalid, err)
		assert.Nil(t, claims)
	})

	t.Run("Wrong Signature", func(t *testing.T) {
		wrongSvc := service.NewAuthService(mockUserRepo, mockTokenRepo, "wrong-secret", tokenLifetime)
		mockUserRepo.On("FindByID", userID).Return(createTestUser(userID), nil).Once()
		wrongToken, err := wrongSvc.Generate(userID)
		require.NoError(t, err)

		claims, err := svc.Validate(wrongToken)
		assert.Error(t, err)
		assert.Equal(t, service.ErrTokenInvalid, err)
		assert.Nil(t, claims)
	})

	t.Run("Expired Token", func(t *testing.T) {
		expiredClaims := service.Claims{
			UserID: userID,
			Email:  "test@example.com",
			Role:   "user",
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        "test-jti",
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
				Subject:   "access_token",
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
		expiredToken, err := token.SignedString([]byte(jwtSecret))
		require.NoError(t, err)

		claims, err := svc.Validate(expiredToken)
		assert.Error(t, err)
		assert.Equal(t, service.ErrTokenExpired, err)
		assert.Nil(t, claims)
	})

	t.Run("Revoked Token", func(t *testing.T) {
		mockTokenRepo.On("IsBlacklisted", tokenID).Return(true, nil).Once()

		claims, err := svc.Validate(validToken)
		assert.Error(t, err)
		assert.Equal(t, service.ErrTokenInvalid, err)
		assert.Nil(t, claims)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Blacklist Check Failed", func(t *testing.T) {
		mockTokenRepo.On("IsBlacklisted", tokenID).Return(false, errors.New("db error")).Once()

		claims, err := svc.Validate(validToken)
		assert.Error(t, err)
		assert.Nil(t, claims)
		mockTokenRepo.AssertExpectations(t)
	})
}

func TestAuthService_IsTokenRevoked(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtSecret, tokenLifetime)

	jti := "test-jwt-id"

	t.Run("Empty Token ID", func(t *testing.T) {
		revoked, err := svc.IsTokenRevoked("")
		assert.NoError(t, err)
		assert.False(t, revoked)
	})

	t.Run("Token Is Revoked", func(t *testing.T) {
		mockTokenRepo.On("IsBlacklisted", jti).Return(true, nil).Once()

		revoked, err := svc.IsTokenRevoked(jti)
		assert.NoError(t, err)
		assert.True(t, revoked)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Token Is Not Revoked", func(t *testing.T) {
		mockTokenRepo.On("IsBlacklisted", jti).Return(false, nil).Once()

		revoked, err := svc.IsTokenRevoked(jti)
		assert.NoError(t, err)
		assert.False(t, revoked)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		mockTokenRepo.On("IsBlacklisted", jti).Return(false, errors.New("db error")).Once()

		revoked, err := svc.IsTokenRevoked(jti)
		assert.Error(t, err)
		assert.Equal(t, service.ErrBlacklistCheckFail, err)
		assert.False(t, revoked)
		mockTokenRepo.AssertExpectations(t)
	})
}

func TestAuthService_FindUserById(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtSecret, tokenLifetime)

	userID := uint(123)

	t.Run("User Found", func(t *testing.T) {
		user := createTestUser(userID)
		expectedDTO := user.ToDTO()

		mockUserRepo.On("FindByID", userID).Return(user, nil).Once()

		userDTO, err := svc.FindUserById(userID)
		assert.NoError(t, err)
		assert.NotNil(t, userDTO)

		assert.Equal(t, expectedDTO.ID, userDTO.ID)
		assert.Equal(t, expectedDTO.Username, userDTO.Username)
		assert.Equal(t, expectedDTO.Email, userDTO.Email)
		assert.Equal(t, expectedDTO.Role, userDTO.Role)

		mockUserRepo.AssertExpectations(t)
	})
	t.Run("User Not Found", func(t *testing.T) {
		mockUserRepo.On("FindByID", userID).Return(nil, errors.New("user not found")).Once()

		userDTO, err := svc.FindUserById(userID)
		assert.Error(t, err)
		assert.Nil(t, userDTO)
		assert.Equal(t, "user not found", err.Error())
		mockUserRepo.AssertExpectations(t)
	})
}

func TestAuthService_Invalidate(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtSecret, tokenLifetime)

	jti := "test-jwt-id"

	t.Run("Empty Token ID", func(t *testing.T) {
		err := svc.Invalidate("")
		assert.Error(t, err)
		assert.Equal(t, service.ErrTokenInvalid, err)
	})

	t.Run("Success", func(t *testing.T) {
		mockTokenRepo.On("Add", mock.MatchedBy(func(token *model.BlacklistedToken) bool {
			return token.JTI == jti
		})).Return(nil).Once()

		err := svc.Invalidate(jti)
		assert.NoError(t, err)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		mockTokenRepo.On("Add", mock.MatchedBy(func(token *model.BlacklistedToken) bool {
			return token.JTI == jti
		})).Return(errors.New("db error")).Once()

		err := svc.Invalidate(jti)
		assert.Error(t, err)
		assert.Equal(t, service.ErrTokenBlacklistFail, err)
		mockTokenRepo.AssertExpectations(t)
	})
}

func TestAuthService_CleanupExpired(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtSecret, tokenLifetime)

	t.Run("Success", func(t *testing.T) {
		mockTokenRepo.On("RemoveExpired").Return(nil).Once()

		err := svc.CleanupExpired()
		assert.NoError(t, err)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		mockTokenRepo.On("RemoveExpired").Return(errors.New("db error")).Once()

		err := svc.CleanupExpired()
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		mockTokenRepo.AssertExpectations(t)
	})
}
