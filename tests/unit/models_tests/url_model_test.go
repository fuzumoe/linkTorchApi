package model_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

func TestURL(t *testing.T) {
	t.Run("To DTO", func(t *testing.T) {
		createdAt := time.Date(2025, 7, 9, 12, 0, 0, 0, time.UTC)
		updatedAt := createdAt.Add(time.Hour)
		url := &model.URL{
			ID:          1,
			UserID:      2,
			OriginalURL: "https://example.com",
			Status:      model.StatusDone,
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
	})

	t.Run("From Create Input", func(t *testing.T) {
		input := &model.CreateURLInput{
			UserID:      2,
			OriginalURL: "https://new-example.com",
		}

		url := model.URLFromCreateInput(input)

		assert.Equal(t, input.UserID, url.UserID, "UserID should match")
		assert.Equal(t, input.OriginalURL, url.OriginalURL, "OriginalURL should match")
		// Expect default status to be "queued".
		assert.Equal(t, model.StatusQueued, url.Status, "Status should default to 'queued'")
		assert.NotZero(t, url.CreatedAt, "CreatedAt should be set")
		assert.NotZero(t, url.UpdatedAt, "UpdatedAt should be set")
	})

	t.Run("Table Name", func(t *testing.T) {
		expected := "urls"
		url := model.URL{}

		assert.Equal(t, expected, url.TableName(), "TableName should return 'urls'")
	})

	t.Run("URL DTO", func(t *testing.T) {
		createdAt := time.Date(2025, 7, 9, 12, 0, 0, 0, time.UTC)
		updatedAt := createdAt.Add(time.Hour)
		dto := &model.URLDTO{
			ID:          1,
			UserID:      2,
			OriginalURL: "https://example.com",
			Status:      model.StatusDone,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}

		assert.Equal(t, uint(1), dto.ID, "ID should be 1")
		assert.Equal(t, uint(2), dto.UserID, "UserID should be 2")
		assert.Equal(t, "https://example.com", dto.OriginalURL, "OriginalURL should be 'https://example.com'")
		assert.Equal(t, model.StatusDone, dto.Status, "Status should be 'done'")
		assert.WithinDuration(t, createdAt, dto.CreatedAt, time.Second, "CreatedAt should match")
		assert.WithinDuration(t, updatedAt, dto.UpdatedAt, time.Second, "UpdatedAt should match")
	})
	t.Run("UpdateURL Valid Input", func(t *testing.T) {
		validJSON := `{"original_url": "https://example.com", "status": "running"}`
		var input model.UpdateURLInput

		// Use Gin binding to validate the JSON.
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString(validJSON))
		ctx.Request.Header.Set("Content-Type", "application/json")
		err := ctx.ShouldBindJSON(&input)
		assert.NoError(t, err, "Valid input should not produce an error")
		assert.Equal(t, "https://example.com", input.OriginalURL)
		assert.Equal(t, "running", input.Status)
	})
	t.Run("UpdateURL Invalid Input", func(t *testing.T) {
		// Invalid URL format and status not one of the allowed values.
		invalidJSON := `{"original_url": "not-a-url", "status": "invalid"}`

		var input model.UpdateURLInput
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString(invalidJSON))
		ctx.Request.Header.Set("Content-Type", "application/json")
		err := ctx.ShouldBindJSON(&input)
		assert.Error(t, err, "Invalid input should produce a validation error")
	})
}
