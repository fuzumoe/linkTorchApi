package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

func TestAnalysisResult(t *testing.T) {
	t.Run("To DTO", func(t *testing.T) {
		createdAt := time.Date(2025, 7, 9, 12, 0, 0, 0, time.UTC)
		updatedAt := createdAt.Add(time.Hour)
		result := &model.AnalysisResult{
			ID:           1,
			URLID:        2,
			HTMLVersion:  "HTML5",
			Title:        "Test Page",
			H1Count:      1,
			H2Count:      2,
			H3Count:      3,
			H4Count:      4,
			H5Count:      5,
			H6Count:      6,
			HasLoginForm: true,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		}

		dto := result.ToDTO()

		assert.Equal(t, result.ID, dto.ID, "ID should match")
		assert.Equal(t, result.URLID, dto.URLID, "URLID should match")
		assert.Equal(t, result.HTMLVersion, dto.HTMLVersion, "HTMLVersion should match")
		assert.Equal(t, result.Title, dto.Title, "Title should match")
		assert.Equal(t, result.H1Count, dto.H1Count, "H1Count should match")
		assert.Equal(t, result.H2Count, dto.H2Count, "H2Count should match")
		assert.Equal(t, result.H3Count, dto.H3Count, "H3Count should match")
		assert.Equal(t, result.H4Count, dto.H4Count, "H4Count should match")
		assert.Equal(t, result.H5Count, dto.H5Count, "H5Count should match")
		assert.Equal(t, result.H6Count, dto.H6Count, "H6Count should match")
		assert.Equal(t, result.HasLoginForm, dto.HasLoginForm, "HasLoginForm should match")
		assert.WithinDuration(t, result.CreatedAt, dto.CreatedAt, time.Second, "CreatedAt should match")
		assert.WithinDuration(t, result.UpdatedAt, dto.UpdatedAt, time.Second, "UpdatedAt should match")
	})

	t.Run("From Create Input", func(t *testing.T) {
		input := &model.CreateAnalysisResultInput{
			URLID:        2,
			HTMLVersion:  "HTML5",
			Title:        "New Test Page",
			H1Count:      1,
			H2Count:      2,
			H3Count:      3,
			H4Count:      4,
			H5Count:      5,
			H6Count:      6,
			HasLoginForm: true,
		}

		result := model.AnalysisResultFromCreateInput(input)

		assert.Equal(t, input.URLID, result.URLID, "URLID should match")
		assert.Equal(t, input.HTMLVersion, result.HTMLVersion, "HTMLVersion should match")
		assert.Equal(t, input.Title, result.Title, "Title should match")
		assert.Equal(t, input.H1Count, result.H1Count, "H1Count should match")
		assert.Equal(t, input.H2Count, result.H2Count, "H2Count should match")
		assert.Equal(t, input.H3Count, result.H3Count, "H3Count should match")
		assert.Equal(t, input.H4Count, result.H4Count, "H4Count should match")
		assert.Equal(t, input.H5Count, result.H5Count, "H5Count should match")
		assert.Equal(t, input.H6Count, result.H6Count, "H6Count should match")
		assert.Equal(t, input.HasLoginForm, result.HasLoginForm, "HasLoginForm should match")
		assert.NotZero(t, result.CreatedAt, "CreatedAt should be set")
		assert.NotZero(t, result.UpdatedAt, "UpdatedAt should be set")
	})

	t.Run("Table Name", func(t *testing.T) {
		expected := "analysis_results"
		result := model.AnalysisResult{}

		assert.Equal(t, expected, result.TableName(), "TableName should return 'analysis_results'")
	})
}
