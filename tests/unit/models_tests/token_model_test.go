package model_test

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

func TestTokenClaims(t *testing.T) {
	claims := model.TokenClaims{
		UserID: 123,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
			IssuedAt:  time.Now().Unix(),
			Subject:   "test-subject",
			Issuer:    "test-issuer",
			Id:        "test-jti",
		},
	}

	assert.Equal(t, uint(123), claims.UserID)
	assert.Equal(t, "test-jti", claims.Id)
	assert.Equal(t, "test-subject", claims.Subject)
	assert.Equal(t, "test-issuer", claims.Issuer)
}

func TestNewJTI(t *testing.T) {
	jti1 := model.NewJTI()
	jti2 := model.NewJTI()

	// JTI should not be empty.
	assert.NotEmpty(t, jti1)

	// Two generated JTIs should be different.
	assert.NotEqual(t, jti1, jti2)

	// Length should be consistent with UUID format.
	assert.Len(t, jti1, 36)
}

func TestBlacklistedToken_TableName(t *testing.T) {
	token := model.BlacklistedToken{}
	assert.Equal(t, "blacklisted_tokens", token.TableName())
}

func TestBlacklistedToken_ToDTO(t *testing.T) {
	now := time.Now()
	token := model.BlacklistedToken{
		ID:        42,
		JTI:       "test-jti-value",
		ExpiresAt: now,
		CreatedAt: now.Add(-time.Hour),
	}

	dto := token.ToDTO()

	// Verify DTO has correct values.
	assert.Equal(t, "test-jti-value", dto.JTI)
	assert.Equal(t, now, dto.ExpiresAt)

}

func TestFromJTI(t *testing.T) {
	now := time.Now()
	jti := "test-jti-creation"

	token := model.FromJTI(jti, now)

	// Verify token has correct values
	assert.Equal(t, jti, token.JTI)
	assert.Equal(t, now, token.ExpiresAt)

	// Other fields should have zero values
	assert.Zero(t, token.ID)
	assert.True(t, token.CreatedAt.IsZero())
	assert.False(t, token.DeletedAt.Valid)
}
