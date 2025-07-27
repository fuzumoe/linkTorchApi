package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

var (
	ErrTokenInvalid       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token is expired")
	ErrTokenBlacklistFail = errors.New("failed to blacklist token")
	ErrBlacklistCheckFail = errors.New("failed to check token blacklist")
)

// Claims defines the JWT claims.
type Claims struct {
	jwt.RegisteredClaims
	UserID uint           `json:"user_id"`
	Email  string         `json:"email"`
	Role   model.UserRole `json:"role"`
}

type AuthService interface {
	AuthenticateBasic(email, password string) (*model.UserDTO, error)
	Validate(token string) (*Claims, error)
	IsTokenRevoked(tokenID string) (bool, error)
	FindUserById(userID uint) (*model.UserDTO, error)
	Generate(userID uint) (string, error)
	Invalidate(tokenID string) error
	CleanupExpired() error
}

type authService struct {
	userRepo    repository.UserRepository
	tokenRepo   repository.TokenRepository
	jwtSecret   string
	jwtLifetime time.Duration
}

func NewAuthService(userRepo repository.UserRepository, tokenRepo repository.TokenRepository, jwtSecret string, jwtLifetime time.Duration) AuthService {
	return &authService{
		userRepo:    userRepo,
		tokenRepo:   tokenRepo,
		jwtSecret:   jwtSecret,
		jwtLifetime: jwtLifetime,
	}
}

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

func (a *authService) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.jwtSecret), nil
	})
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

	revoked, err := a.IsTokenRevoked(claims.ID)
	if err != nil {
		return nil, err
	}
	if revoked {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

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

func (a *authService) FindUserById(userID uint) (*model.UserDTO, error) {
	user, err := a.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	return user.ToDTO(), nil
}

func (a *authService) Generate(userID uint) (string, error) {
	user, err := a.userRepo.FindByID(userID)
	if err != nil {
		return "", err
	}

	expirationTime := time.Now().Add(a.jwtLifetime)
	claims := &Claims{
		UserID: userID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        generateTokenID(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(a.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
func (a *authService) Invalidate(tokenID string) error {

	if tokenID == "" {
		return ErrTokenInvalid
	}

	blacklistedToken := &model.BlacklistedToken{
		JTI:       tokenID,
		ExpiresAt: time.Now().Add(a.jwtLifetime),
	}

	err := a.tokenRepo.Add(blacklistedToken)
	if err != nil {
		return ErrTokenBlacklistFail
	}

	return nil
}

func (a *authService) CleanupExpired() error {
	return a.tokenRepo.RemoveExpired()
}

func generateTokenID() string {
	return uuid.New().String()
}
