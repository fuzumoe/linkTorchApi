package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

// TestURLToDTO tests the conversion of URL model to URLDTO.
func TestURLToDTO(t *testing.T) {
	createdAt := time.Date(2025, 7, 9, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	url := &model.URL{
		ID:          1,
		UserID:      2,
		OriginalURL: "https://example.com",
		Status:      "done",
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	dto := url.ToDTO()

	assert.Equal(t, url.ID, dto.ID, "ID should match")
	assert.Equal(t, url.UserID, dto.UserID, "UserID should match")
	assert.Equal(t, url.OriginalURL, dto.OriginalURL, "OriginalURL should match")
	assert.Equal(t, url.Status, dto.Status, "Status should match")
	assert.WithinDuration(t, url.CreatedAt, dto.CreatedAt, time.Second, "CreatedAt should match")
	assert.WithinDuration(t, url.UpdatedAt, dto.UpdatedAt, time.Second, "UpdatedAt should match")
}

// TestURLFromCreateInput tests the conversion from CreateURLInput to URL model.
func TestURLFromCreateInput(t *testing.T) {
	input := &model.CreateURLInput{
		UserID:      2,
		OriginalURL: "https://new-example.com",
	}

	url := model.URLFromCreateInput(input)

	assert.Equal(t, input.UserID, url.UserID, "UserID should match")
	assert.Equal(t, input.OriginalURL, url.OriginalURL, "OriginalURL should match")
	assert.Equal(t, "pending", url.Status, "Status should default to 'queued'")
	assert.NotZero(t, url.CreatedAt, "CreatedAt should be set")
	assert.NotZero(t, url.UpdatedAt, "UpdatedAt should be set")

}

// TestURLTableName tests the TableName method of the URL model.
func TestURLTableName(t *testing.T) {
	expected := "urls"
	url := model.URL{}

	assert.Equal(t, expected, url.TableName(), "TableName should return 'urls'")

}

// TestURLDTO tests the URLDTO struct.
func TestURLDTO(t *testing.T) {
	createdAt := time.Date(2025, 7, 9, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	dto := &model.URLDTO{
		ID:          1,
		UserID:      2,
		OriginalURL: "https://example.com",
		Status:      "done",
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	// Fix the order of expected vs actual and make sure types match
	assert.Equal(t, uint(1), dto.ID, "ID should be 1")
	assert.Equal(t, uint(2), dto.UserID, "UserID should be 2")
	assert.Equal(t, "https://example.com", dto.OriginalURL, "OriginalURL should be 'https://example.com'")
	assert.Equal(t, "done", dto.Status, "Status should be 'done'")
	assert.WithinDuration(t, createdAt, dto.CreatedAt, time.Second, "CreatedAt should match")
	assert.WithinDuration(t, updatedAt, dto.UpdatedAt, time.Second, "UpdatedAt should match")
}
