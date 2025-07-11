package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

var (
	ErrTokenInvalid       = errors.New("token is invalid")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenBlacklistFail = errors.New("failed to blacklist token")
	ErrBlacklistCheckFail = errors.New("failed to check blacklist")
)

// JWTClaims extends standard JWT claims with our custom fields
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID uint `json:"user_id"`
}

// TokenService defines methods for JWT operations
type TokenService interface {
	Generate(userID uint) (string, error)
	Validate(tokenString string) (*JWTClaims, error)
	Invalidate(tokenID string) error
	IsBlacklisted(tokenID string) (bool, error)
	CleanupExpired() error
}

// UserLookup defines the minimal user methods needed by AuthService
type UserLookup interface {
	FindByID(id uint) (*model.User, error)
}

// AuthService implements TokenService only
type AuthService struct {
	userLookup  UserLookup
	tokenRepo   repository.TokenRepository
	jwtSecret   string
	tokenExpiry time.Duration
}

// NewAuthService creates a new AuthService
func NewAuthService(
	userLookup UserLookup,
	tokenRepo repository.TokenRepository,
	jwtSecret string,
	tokenExpiry time.Duration,
) *AuthService {
	return &AuthService{
		userLookup:  userLookup,
		tokenRepo:   tokenRepo,
		jwtSecret:   jwtSecret,
		tokenExpiry: tokenExpiry,
	}
}

// Generate creates a new JWT token for a user
func (s *AuthService) Generate(userID uint) (string, error) {
	// First check if user exists
	if _, err := s.userLookup.FindByID(userID); err != nil {
		return "", errors.New("user not found")
	}

	tokenID := uuid.New().String()
	now := time.Now()

	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenExpiry)),
			Subject:   "access_token",
		},
		UserID: userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// Validate checks if a token is valid and returns its claims
func (s *AuthService) Validate(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

// Invalidate adds a token to the blacklist
func (s *AuthService) Invalidate(tokenID string) error {
	err := s.tokenRepo.Add(&model.BlacklistedToken{
		JTI:       tokenID,
		ExpiresAt: time.Now().Add(s.tokenExpiry),
	})

	if err != nil {
		return ErrTokenBlacklistFail
	}
	return nil
}

// IsBlacklisted checks if a token is in the blacklist
func (s *AuthService) IsBlacklisted(tokenID string) (bool, error) {
	isBlacklisted, err := s.tokenRepo.IsBlacklisted(tokenID)
	if err != nil {
		return false, ErrBlacklistCheckFail
	}
	return isBlacklisted, nil
}

// CleanupExpired removes expired tokens from the blacklist
func (s *AuthService) CleanupExpired() error {
	return s.tokenRepo.RemoveExpired()
}
