// internal/service/token_service.go
package service

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// TokenService defines JWT generation, validation, and revocation.
type TokenService interface {
	// Generate creates a new JWT for the given user ID, records its JTI in the blacklist store,
	// and returns the signed token string.
	Generate(userID uint) (string, error)

	// Validate parses and validates a JWT string, returning its claims.
	Validate(tokenStr string) (*model.TokenClaims, error)

	// Blacklist revokes the given JTI immediately.
	Blacklist(jti string, expiresAt time.Time) error

	// IsBlacklisted returns true if the given JTI has been revoked and not yet expired.
	IsBlacklisted(jti string) (bool, error)
}

type tokenService struct {
	secret   string
	lifetime time.Duration
	repo     repository.TokenRepository
}

// NewTokenService constructs a TokenService.
//
//	secret   – HMAC signing secret
//	lifetime – token TTL (e.g. 24h)
//	repo     – repository for storing/revoking JTIs
func NewTokenService(
	secret string,
	lifetime time.Duration,
	repo repository.TokenRepository,
) TokenService {
	return &tokenService{
		secret:   secret,
		lifetime: lifetime,
		repo:     repo,
	}
}

func (s *tokenService) Generate(userID uint) (string, error) {
	// 1) Create unique JTI
	jti := model.NewJTI()

	// 2) Build claims
	now := time.Now().UTC()
	exp := now.Add(s.lifetime)
	claims := model.TokenClaims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			Id:        jti,
			IssuedAt:  now.Unix(),
			ExpiresAt: exp.Unix(),
		},
	}

	// 3) Sign token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.secret))
	if err != nil {
		return "", err
	}

	// 4) Record in blacklist store (so we can revoke later), with same expiry
	if err := s.repo.Add(&model.BlacklistedToken{
		JTI:       jti,
		ExpiresAt: exp,
	}); err != nil {
		return "", err
	}

	return signed, nil
}

func (s *tokenService) Validate(tokenStr string) (*model.TokenClaims, error) {
	// 1) Parse with claims
	token, err := jwt.ParseWithClaims(tokenStr, &model.TokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	// 2) Extract typed claims
	claims, ok := token.Claims.(*model.TokenClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

func (s *tokenService) Blacklist(jti string, expiresAt time.Time) error {
	// Immediately mark this JTI as revoked.
	return s.repo.Add(&model.BlacklistedToken{
		JTI:       jti,
		ExpiresAt: expiresAt,
	})
}

func (s *tokenService) IsBlacklisted(jti string) (bool, error) {
	return s.repo.IsBlacklisted(jti)
}
