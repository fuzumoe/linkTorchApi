package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

func TestNewJTI(t *testing.T) {
	jti1 := model.NewJTI()
	jti2 := model.NewJTI()

	assert.NotEmpty(t, jti1)
	assert.NotEqual(t, jti1, jti2)
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

	assert.Equal(t, "test-jti-value", dto.JTI)
	assert.Equal(t, now, dto.ExpiresAt)
}

func TestFromJTI(t *testing.T) {
	now := time.Now()
	jti := "test-jti-creation"

	token := model.FromJTI(jti, now)
	assert.Equal(t, jti, token.JTI)
	assert.Equal(t, now, token.ExpiresAt)
	assert.Zero(t, token.ID)
	assert.True(t, token.CreatedAt.IsZero())
	assert.False(t, token.DeletedAt.Valid)
}
