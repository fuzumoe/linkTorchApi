package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// Error constants for token validation and management.
var (
	ErrTokenInvalid       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token is expired")
	ErrTokenBlacklistFail = errors.New("failed to blacklist token")
	ErrBlacklistCheckFail = errors.New("failed to check token blacklist")
)

// Claims defines the JWT claims.
type Claims struct {
	jwt.RegisteredClaims
	UserID uint `json:"user_id"`
}

// AuthService defines authentication operations.
type AuthService interface {
	// AuthenticateBasic validates a user's email and password.
	AuthenticateBasic(email, password string) (*model.UserDTO, error)
	// Validate parses and validates a JWT token.
	Validate(token string) (*Claims, error)
	// IsTokenRevoked checks whether the token with the given ID was revoked.
	IsTokenRevoked(tokenID string) (bool, error)
	// FindUserById retrieves a user by its ID.
	FindUserById(userID uint) (*model.UserDTO, error)
	// Generate creates a new JWT token for the given user ID.
	Generate(userID uint) (string, error)
	// Invalidate invalidates a token given its ID.
	Invalidate(tokenID string) error
	// CleanupExpired removes expired tokens from the blacklist.
	CleanupExpired() error
}

type authService struct {
	userRepo    repository.UserRepository
	tokenRepo   repository.TokenRepository
	jwtSecret   string
	jwtLifetime time.Duration
}

// NewAuthService constructs a new AuthService.
func NewAuthService(userRepo repository.UserRepository, tokenRepo repository.TokenRepository, jwtSecret string, jwtLifetime time.Duration) AuthService {
	return &authService{
		userRepo:    userRepo,
		tokenRepo:   tokenRepo,
		jwtSecret:   jwtSecret,
		jwtLifetime: jwtLifetime,
	}
}

// AuthenticateBasic validates a user's email and password using bcrypt.
func (a *authService) AuthenticateBasic(email, password string) (*model.UserDTO, error) {
	user, err := a.userRepo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return nil, errors.New("invalid credentials")
	}
	return user.ToDTO(), nil
}

// Validate parses a JWT token string and returns its claims.
func (a *authService) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.jwtSecret), nil
	})

	// Handle parsing errors
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	// Check if the token has been revoked.
	revoked, err := a.IsTokenRevoked(claims.ID)
	if err != nil {
		return nil, err
	}
	if revoked {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// IsTokenRevoked checks if a token has been revoked by checking if it's in the blacklist.
func (a *authService) IsTokenRevoked(tokenID string) (bool, error) {
	// If token ID is empty, it can't be in the blacklist.
	if tokenID == "" {
		return false, nil
	}

	// Check if the token is in the blacklist.
	isBlacklisted, err := a.tokenRepo.IsBlacklisted(tokenID)
	if err != nil {
		return false, ErrBlacklistCheckFail
	}

	return isBlacklisted, nil
}

// FindUserById retrieves a user by its ID.
func (a *authService) FindUserById(userID uint) (*model.UserDTO, error) {
	user, err := a.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	return user.ToDTO(), nil
}

// Generate creates a new JWT token for the specified user ID.
func (a *authService) Generate(userID uint) (string, error) {
	// Verify that the user exists
	_, err := a.userRepo.FindByID(userID)
	if err != nil {
		return "", err
	}

	expirationTime := time.Now().Add(a.jwtLifetime)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        generateTokenID(), // Generate a unique token ID.
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(a.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Invalidate adds a token to the blacklist to invalidate it.
func (a *authService) Invalidate(tokenID string) error {
	// If token ID is empty, we can't blacklist it.
	if tokenID == "" {
		return ErrTokenInvalid
	}

	// Create a new blacklisted token
	blacklistedToken := &model.BlacklistedToken{
		JTI:       tokenID,
		ExpiresAt: time.Now().Add(a.jwtLifetime),
	}

	// Add the token to the blacklist.
	err := a.tokenRepo.Add(blacklistedToken)
	if err != nil {
		return ErrTokenBlacklistFail
	}

	return nil
}

// CleanupExpired removes expired tokens from the blacklist.
func (a *authService) CleanupExpired() error {
	return a.tokenRepo.RemoveExpired()
}

// Helper function to generate a unique token ID.
func generateTokenID() string {
	// return fmt.Sprintf("%d", time.Now().UnixNano())
	return uuid.New().String()
}
