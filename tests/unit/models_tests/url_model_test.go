package model_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

func TestURL(t *testing.T) {
	t.Run("To DTO", func(t *testing.T) {
		createdAt := time.Date(2025, 7, 9, 12, 0, 0, 0, time.UTC)
		updatedAt := createdAt.Add(time.Hour)
		u := &model.URL{
			ID:          1,
			UserID:      2,
			OriginalURL: "https://example.com",
			Status:      model.StatusDone,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}

		dto := u.ToDTO()

		assert.Equal(t, u.ID, dto.ID, "ID should match")
		assert.Equal(t, u.UserID, dto.UserID, "UserID should match")
		assert.Equal(t, u.OriginalURL, dto.OriginalURL, "OriginalURL should match")
		assert.Equal(t, u.Status, dto.Status, "Status should match")
		assert.WithinDuration(t, u.CreatedAt, dto.CreatedAt, time.Second, "CreatedAt should match")
		assert.WithinDuration(t, u.UpdatedAt, dto.UpdatedAt, time.Second, "UpdatedAt should match")
	})

	t.Run("PaginationMetaDTO", func(t *testing.T) {
		meta := model.PaginationMetaDTO{
			Page:       2,
			PageSize:   10,
			TotalItems: 25,
			TotalPages: 3,
		}

		assert.Equal(t, 2, meta.Page, "Page should be 2")
		assert.Equal(t, 10, meta.PageSize, "PageSize should be 10")
		assert.Equal(t, 25, meta.TotalItems, "TotalItems should be 25")
		assert.Equal(t, 3, meta.TotalPages, "TotalPages should be 3")
	})

	t.Run("PaginatedResponse", func(t *testing.T) {
		dtos := []model.URLDTO{
			{
				ID:          1,
				UserID:      2,
				OriginalURL: "https://example1.com",
				Status:      model.StatusDone,
			},
			{
				ID:          2,
				UserID:      2,
				OriginalURL: "https://example2.com",
				Status:      model.StatusQueued,
			},
		}

		paginatedResponse := model.PaginatedResponse[model.URLDTO]{
			Data: dtos,
			Pagination: model.PaginationMetaDTO{
				Page:       1,
				PageSize:   10,
				TotalItems: 2,
				TotalPages: 1,
			},
		}

		assert.Len(t, paginatedResponse.Data, 2, "Should have 2 items in Data")
		assert.Equal(t, uint(1), paginatedResponse.Data[0].ID, "First item ID should be 1")
		assert.Equal(t, uint(2), paginatedResponse.Data[1].ID, "Second item ID should be 2")
		assert.Equal(t, 1, paginatedResponse.Pagination.Page, "Page should be 1")
		assert.Equal(t, 10, paginatedResponse.Pagination.PageSize, "PageSize should be 10")
		assert.Equal(t, 2, paginatedResponse.Pagination.TotalItems, "TotalItems should be 2")
		assert.Equal(t, 1, paginatedResponse.Pagination.TotalPages, "TotalPages should be 1")
	})

	t.Run("PaginatedResponse JSON Marshaling", func(t *testing.T) {
		dtos := []model.URLDTO{
			{
				ID:          1,
				UserID:      2,
				OriginalURL: "https://example1.com",
				Status:      model.StatusDone,
			},
		}

		paginatedResponse := model.PaginatedResponse[model.URLDTO]{
			Data: dtos,
			Pagination: model.PaginationMetaDTO{
				Page:       1,
				PageSize:   10,
				TotalItems: 1,
				TotalPages: 1,
			},
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(paginatedResponse)
		require.NoError(t, err, "Marshaling should not produce an error")

		// Unmarshal back to struct
		var unmarshaled map[string]interface{}
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err, "Unmarshaling should not produce an error")

		// Check structure
		data, ok := unmarshaled["data"].([]interface{})
		require.True(t, ok, "Should have 'data' array field")
		require.Len(t, data, 1, "Data should have 1 item")

		pagination, ok := unmarshaled["pagination"].(map[string]interface{})
		require.True(t, ok, "Should have 'pagination' object field")
		assert.Equal(t, float64(1), pagination["page"], "page should be 1")
		assert.Equal(t, float64(10), pagination["pageSize"], "pageSize should be 10")
		assert.Equal(t, float64(1), pagination["totalItems"], "totalItems should be 1")
		assert.Equal(t, float64(1), pagination["totalPages"], "totalPages should be 1")
	})
	t.Run("From Create Input", func(t *testing.T) {
		input := &model.CreateURLInputDTO{
			UserID:      2,
			OriginalURL: "https://new-example.com",
		}

		u := model.URLFromCreateInput(input)

		assert.Equal(t, input.UserID, u.UserID, "UserID should match")
		assert.Equal(t, input.OriginalURL, u.OriginalURL, "OriginalURL should match")
		// Expect default status to be "queued".
		assert.Equal(t, model.StatusQueued, u.Status, "Status should default to 'queued'")
		assert.NotZero(t, u.CreatedAt, "CreatedAt should be set")
		assert.NotZero(t, u.UpdatedAt, "UpdatedAt should be set")
	})

	t.Run("Table Name", func(t *testing.T) {
		expected := "urls"
		u := model.URL{}

		assert.Equal(t, expected, u.TableName(), "TableName should return 'urls'")
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

		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString(validJSON))
		ctx.Request.Header.Set("Content-Type", "application/json")
		err := ctx.ShouldBindJSON(&input)
		assert.NoError(t, err, "Valid input should not produce an error")
		assert.Equal(t, "https://example.com", input.OriginalURL)
		assert.Equal(t, "running", input.Status)
	})

	t.Run("UpdateURL Invalid Input", func(t *testing.T) {
		invalidJSON := `{"original_url": "not-a-url", "status": "invalid"}`
		var input model.UpdateURLInput

		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString(invalidJSON))
		ctx.Request.Header.Set("Content-Type", "application/json")
		err := ctx.ShouldBindJSON(&input)
		assert.Error(t, err, "Invalid input should produce a validation error")
	})

	t.Run("Parsed URL", func(t *testing.T) {
		u := &model.URL{
			OriginalURL: "https://example.com/path?query=value",
		}
		parsed := u.URL()
		require.NotNil(t, parsed, "Parsed URL should not be nil")
		assert.Equal(t, "example.com", parsed.Host, "Host should be 'example.com'")
		assert.Equal(t, "https", parsed.Scheme, "Scheme should be 'https'")
		assert.Equal(t, "/path", parsed.Path, "Path should be '/path'")
	})

	// New tests for JSON unmarshaling of AnalysisResult and Link

	t.Run("AnalysisResult JSON", func(t *testing.T) {
		// Adjust the JSON payload to match your AnalysisResult struct.
		// Ensure that has_login_form is a proper JSON boolean.
		jsonStr := `{
            "id": 1,
            "url_id": 1,
            "html_version": "HTML5",
            "title": "Test Title",
            "h1_count": 2,
            "h2_count": 3,
            "h3_count": 4,
            "h4_count": 0,
            "h5_count": 0,
            "h6_count": 0,
            "has_login_form": true,
            "internal_link_count": 5,
            "external_link_count": 6,
            "broken_link_count": 0,
            "created_at": "2025-07-09T12:00:00Z",
            "updated_at": "2025-07-09T13:00:00Z"
        }`
		var ar model.AnalysisResult
		err := json.Unmarshal([]byte(jsonStr), &ar)
		require.NoError(t, err, "AnalysisResult should unmarshal without error")
		assert.True(t, ar.HasLoginForm, "HasLoginForm should be true")
	})

	t.Run("Link JSON", func(t *testing.T) {
		// Adjust the JSON payload to match your Link struct.
		// Ensure that is_external is a proper JSON boolean.
		jsonStr := `{
            "id": 1,
            "url_id": 1,
            "href": "https://example.com",
            "is_external": false,
            "status_code": 200
        }`
		var l model.Link
		err := json.Unmarshal([]byte(jsonStr), &l)
		require.NoError(t, err, "Link should unmarshal without error")
		assert.False(t, l.IsExternal, "IsExternal should be false")
	})
}
