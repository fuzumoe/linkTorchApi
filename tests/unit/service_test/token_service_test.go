package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
)

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

func (m *MockTokenRepository) CleanExpired() error {
	args := m.Called()
	return args.Error(0)
}

// Add the missing RemoveExpired method to satisfy the TokenRepository interface
func (m *MockTokenRepository) RemoveExpired() error {
	args := m.Called()
	return args.Error(0)
}

func TestTokenService_Generate(t *testing.T) {
	// Setup
	mockRepo := new(MockTokenRepository)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewTokenService(jwtSecret, tokenLifetime, mockRepo)

	userID := uint(123)

	t.Run("Success", func(t *testing.T) {
		// Setup expectations
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
		token, err := jwt.ParseWithClaims(tokenString, &model.TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		require.NoError(t, err)

		claims, ok := token.Claims.(*model.TokenClaims)
		require.True(t, ok)
		assert.Equal(t, userID, claims.UserID)
		assert.NotEmpty(t, claims.Id) // JTI

		// Verify times
		now := time.Now().UTC().Unix()
		assert.LessOrEqual(t, claims.IssuedAt, now)
		assert.Greater(t, claims.ExpiresAt, now)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		// Setup expectations
		mockRepo.On("Add", mock.AnythingOfType("*model.BlacklistedToken")).Return(errors.New("db error")).Once()

		// Execute
		tokenString, err := svc.Generate(userID)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		assert.Empty(t, tokenString)

		mockRepo.AssertExpectations(t)
	})
}

func TestTokenService_Validate(t *testing.T) {
	// Setup
	mockRepo := new(MockTokenRepository)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewTokenService(jwtSecret, tokenLifetime, mockRepo)

	userID := uint(123)

	// Generate a valid token for testing
	mockRepo.On("Add", mock.AnythingOfType("*model.BlacklistedToken")).Return(nil).Once()
	validToken, err := svc.Generate(userID)
	require.NoError(t, err)

	// Parse it to get the claims for later tests
	parsedToken, _ := jwt.ParseWithClaims(validToken, &model.TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	validClaims := parsedToken.Claims.(*model.TokenClaims)

	t.Run("Valid Token", func(t *testing.T) {
		// Execute
		claims, err := svc.Validate(validToken)

		// Verify
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, validClaims.Id, claims.Id)
	})

	t.Run("Invalid Token Format", func(t *testing.T) {
		// Execute
		claims, err := svc.Validate("invalid-token-format")

		// Verify
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("Wrong Signature", func(t *testing.T) {
		// Generate a token with different secret
		wrongSvc := service.NewTokenService("wrong-secret", tokenLifetime, mockRepo)
		mockRepo.On("Add", mock.AnythingOfType("*model.BlacklistedToken")).Return(nil).Once()
		wrongToken, err := wrongSvc.Generate(userID)
		require.NoError(t, err)

		// Try to validate with original service
		claims, err := svc.Validate(wrongToken)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("Expired Token", func(t *testing.T) {
		// Create expired token
		expiredClaims := model.TokenClaims{
			UserID: userID,
			StandardClaims: jwt.StandardClaims{
				Id:        model.NewJTI(),
				IssuedAt:  time.Now().Add(-2 * time.Hour).Unix(),
				ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
		expiredToken, err := token.SignedString([]byte(jwtSecret))
		require.NoError(t, err)

		// Execute
		claims, err := svc.Validate(expiredToken)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestTokenService_Blacklist(t *testing.T) {
	// Setup
	mockRepo := new(MockTokenRepository)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewTokenService(jwtSecret, tokenLifetime, mockRepo)

	jti := model.NewJTI()
	expiresAt := time.Now().Add(tokenLifetime)

	t.Run("Success", func(t *testing.T) {
		// Setup expectations
		mockRepo.On("Add", mock.MatchedBy(func(token *model.BlacklistedToken) bool {
			return token.JTI == jti && token.ExpiresAt.Equal(expiresAt)
		})).Return(nil).Once()

		// Execute
		err := svc.Blacklist(jti, expiresAt)

		// Verify
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		// Setup expectations
		mockRepo.On("Add", mock.MatchedBy(func(token *model.BlacklistedToken) bool {
			return token.JTI == jti && token.ExpiresAt.Equal(expiresAt)
		})).Return(errors.New("db error")).Once()

		// Execute
		err := svc.Blacklist(jti, expiresAt)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		mockRepo.AssertExpectations(t)
	})
}

func TestTokenService_IsBlacklisted(t *testing.T) {
	// Setup
	mockRepo := new(MockTokenRepository)
	jwtSecret := "test-secret-key"
	tokenLifetime := 1 * time.Hour
	svc := service.NewTokenService(jwtSecret, tokenLifetime, mockRepo)

	jti := model.NewJTI()

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
		assert.Equal(t, "db error", err.Error())
		assert.False(t, isBlacklisted)
		mockRepo.AssertExpectations(t)
	})
}
