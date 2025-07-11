package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
)

// MockUserLookup is a mock of the UserLookup interface
type MockUserLookup struct {
	mock.Mock
}

func (m *MockUserLookup) FindByID(id uint) (*model.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

// MockTokenRepository is a mock of the TokenRepository
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

func TestAuthService_Generate(t *testing.T) {
	// Setup
	mockRepo := new(MockTokenRepository)
	mockUserLookup := new(MockUserLookup)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewAuthService(mockUserLookup, mockRepo, jwtSecret, tokenLifetime)

	userID := uint(123)

	t.Run("Success", func(t *testing.T) {
		// Setup expectations for user lookup
		mockUserLookup.On("FindByID", userID).Return(&model.User{ID: userID}, nil).Once()

		// Setup expectations for token repository
		mockRepo.On("Add", mock.AnythingOfType("*model.BlacklistedToken")).Run(func(args mock.Arguments) {
			token := args.Get(0).(*model.BlacklistedToken)
			assert.NotEmpty(t, token.JTI)

			// Verify expiry time is approximately tokenLifetime in the future
			now := time.Now().UTC()
			expectedExpiry := now.Add(tokenLifetime)
			assert.WithinDuration(t, expectedExpiry, token.ExpiresAt, 2*time.Second)
		}).Return(nil).Once()

		// Execute
		tokenString, err := svc.Generate(userID)

		// Verify
		require.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		// Parse the token to verify its contents
		token, err := jwt.ParseWithClaims(tokenString, &service.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		require.NoError(t, err)

		claims, ok := token.Claims.(*service.JWTClaims)
		require.True(t, ok)
		assert.Equal(t, userID, claims.UserID)
		assert.NotEmpty(t, claims.ID) // JTI

		// Verify times
		now := time.Now().UTC()
		assert.WithinDuration(t, now, claims.IssuedAt.Time, 2*time.Second)
		assert.WithinDuration(t, now.Add(tokenLifetime), claims.ExpiresAt.Time, 2*time.Second)

		mockRepo.AssertExpectations(t)
		mockUserLookup.AssertExpectations(t)
	})

	t.Run("User Not Found", func(t *testing.T) {
		// Setup expectations for user lookup to fail
		mockUserLookup.On("FindByID", userID).Return(nil, errors.New("user not found")).Once()

		// Execute
		tokenString, err := svc.Generate(userID)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, "user not found", err.Error())
		assert.Empty(t, tokenString)

		mockUserLookup.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		// Setup expectations for user lookup to succeed
		mockUserLookup.On("FindByID", userID).Return(&model.User{ID: userID}, nil).Once()

		// Setup expectations for token repository to fail
		mockRepo.On("Add", mock.AnythingOfType("*model.BlacklistedToken")).Return(errors.New("db error")).Once()

		// Execute
		tokenString, err := svc.Generate(userID)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		assert.Empty(t, tokenString)

		mockRepo.AssertExpectations(t)
		mockUserLookup.AssertExpectations(t)
	})
}

func TestAuthService_Validate(t *testing.T) {
	// Setup
	mockRepo := new(MockTokenRepository)
	mockUserLookup := new(MockUserLookup)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewAuthService(mockUserLookup, mockRepo, jwtSecret, tokenLifetime)

	userID := uint(123)

	// Generate a valid token for testing
	mockUserLookup.On("FindByID", userID).Return(&model.User{ID: userID}, nil).Once()
	mockRepo.On("Add", mock.AnythingOfType("*model.BlacklistedToken")).Return(nil).Once()
	validToken, err := svc.Generate(userID)
	require.NoError(t, err)

	t.Run("Valid Token", func(t *testing.T) {
		// Execute
		claims, err := svc.Validate(validToken)

		// Verify
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.NotEmpty(t, claims.ID)
	})

	t.Run("Invalid Token Format", func(t *testing.T) {
		// Execute
		claims, err := svc.Validate("invalid-token-format")

		// Verify
		assert.Error(t, err)
		assert.Equal(t, service.ErrTokenInvalid, err)
		assert.Nil(t, claims)
	})

	t.Run("Wrong Signature", func(t *testing.T) {
		// Generate a token with different secret
		wrongSvc := service.NewAuthService(mockUserLookup, mockRepo, "wrong-secret", tokenLifetime)
		mockUserLookup.On("FindByID", userID).Return(&model.User{ID: userID}, nil).Once()
		mockRepo.On("Add", mock.AnythingOfType("*model.BlacklistedToken")).Return(nil).Once()
		wrongToken, err := wrongSvc.Generate(userID)
		require.NoError(t, err)

		// Try to validate with original service
		claims, err := svc.Validate(wrongToken)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, service.ErrTokenInvalid, err)
		assert.Nil(t, claims)
	})

	t.Run("Expired Token", func(t *testing.T) {
		// Create expired token
		expiredClaims := service.JWTClaims{
			UserID: userID,
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        "test-jti",
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
				Subject:   "access_token",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
		expiredToken, err := token.SignedString([]byte(jwtSecret))
		require.NoError(t, err)

		// Execute
		claims, err := svc.Validate(expiredToken)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, service.ErrTokenExpired, err)
		assert.Nil(t, claims)
	})
}

func TestAuthService_Invalidate(t *testing.T) {
	// Setup
	mockRepo := new(MockTokenRepository)
	mockUserLookup := new(MockUserLookup)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewAuthService(mockUserLookup, mockRepo, jwtSecret, tokenLifetime)

	jti := "test-jwt-id"

	t.Run("Success", func(t *testing.T) {
		// Setup expectations
		mockRepo.On("Add", mock.MatchedBy(func(token *model.BlacklistedToken) bool {
			return token.JTI == jti
		})).Return(nil).Once()

		// Execute
		err := svc.Invalidate(jti)

		// Verify
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		// Setup expectations
		mockRepo.On("Add", mock.MatchedBy(func(token *model.BlacklistedToken) bool {
			return token.JTI == jti
		})).Return(errors.New("db error")).Once()

		// Execute
		err := svc.Invalidate(jti)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, service.ErrTokenBlacklistFail, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_IsBlacklisted(t *testing.T) {
	// Setup
	mockRepo := new(MockTokenRepository)
	mockUserLookup := new(MockUserLookup)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewAuthService(mockUserLookup, mockRepo, jwtSecret, tokenLifetime)

	jti := "test-jwt-id"

	t.Run("Token Is Blacklisted", func(t *testing.T) {
		// Setup expectations
		mockRepo.On("IsBlacklisted", jti).Return(true, nil).Once()

		// Execute
		isBlacklisted, err := svc.IsBlacklisted(jti)

		// Verify
		assert.NoError(t, err)
		assert.True(t, isBlacklisted)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Token Is Not Blacklisted", func(t *testing.T) {
		// Setup expectations
		mockRepo.On("IsBlacklisted", jti).Return(false, nil).Once()

		// Execute
		isBlacklisted, err := svc.IsBlacklisted(jti)

		// Verify
		assert.NoError(t, err)
		assert.False(t, isBlacklisted)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		// Setup expectations
		mockRepo.On("IsBlacklisted", jti).Return(false, errors.New("db error")).Once()

		// Execute
		isBlacklisted, err := svc.IsBlacklisted(jti)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, service.ErrBlacklistCheckFail, err)
		assert.False(t, isBlacklisted)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_CleanupExpired(t *testing.T) {
	// Setup
	mockRepo := new(MockTokenRepository)
	mockUserLookup := new(MockUserLookup)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewAuthService(mockUserLookup, mockRepo, jwtSecret, tokenLifetime)

	t.Run("Success", func(t *testing.T) {
		// Setup expectations
		mockRepo.On("RemoveExpired").Return(nil).Once()

		// Execute
		err := svc.CleanupExpired()

		// Verify
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		// Setup expectations
		mockRepo.On("RemoveExpired").Return(errors.New("db error")).Once()

		// Execute
		err := svc.CleanupExpired()

		// Verify
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		mockRepo.AssertExpectations(t)
	})
}
