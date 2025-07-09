package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

// TestBlacklistedTokenToDTO tests the conversion of BlacklistedToken model to BlacklistedTokenDTO.
func TestBlacklistedTokenToDTO(t *testing.T) {
	// Create a sample BlacklistedToken instance
	token := &model.BlacklistedToken{
		ID:        1,
		JTI:       "sample-jti",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	// Convert to DTO
	dto := token.ToDTO()

	// Validate the conversion
	assert.Equal(t, token.JTI, dto.JTI, "JTI should match")
	assert.WithinDuration(t, token.ExpiresAt, dto.ExpiresAt, time.Second, "ExpiresAt should match")
	assert.WithinDuration(t, token.CreatedAt, dto.CreatedAt, time.Second, "CreatedAt should match")
}

// TestBlacklistedTokenFromCreateInput tests the conversion from CreateBlacklistedTokenInput to BlacklistedToken model.
func TestBlacklistedTokenFromCreateInput(t *testing.T) {
	// Create a sample CreateBlacklistedTokenInput instance
	input := &model.CreateBlacklistedTokenInput{
		JTI:       "new-jti",
		ExpiresAt: time.Now().Add(48 * time.Hour),
	}

	// Convert to BlacklistedToken model
	token := model.BlacklistedTokenFromCreateInput(input)

	// Validate the conversion
	assert.Equal(t, input.JTI, token.JTI, "JTI should match")
	assert.WithinDuration(t, input.ExpiresAt, token.ExpiresAt, time.Second, "ExpiresAt should match")
	assert.NotZero(t, token.CreatedAt, "CreatedAt should be set")
}

// TestCreateBlacklistedTokenInput tests the CreateBlacklistedTokenInput struct.
func TestCreateBlacklistedTokenInput(t *testing.T) {
	// Create a sample CreateBlacklistedTokenInput instance
	input := &model.CreateBlacklistedTokenInput{
		JTI:       "test-jti",
		ExpiresAt: time.Now().Add(72 * time.Hour),
	}

	// Validate the input fields
	assert.NotEmpty(t, input.JTI, "JTI should not be empty")
	assert.WithinDuration(t, input.ExpiresAt, time.Now().Add(72*time.Hour), time.Second, "ExpiresAt should be set correctly")
}

// TestBlacklistedTokenTableName tests the TableName method of the BlacklistedToken model.
func TestBlacklistedTokenTableName(t *testing.T) {
	expected := "blacklisted_tokens"
	token := model.BlacklistedToken{}

	// Validate the table name
	assert.Equal(t, expected, token.TableName(), "TableName should return 'blacklisted_tokens'")
}
