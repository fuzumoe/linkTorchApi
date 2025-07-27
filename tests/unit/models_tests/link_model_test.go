package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

func TestLink(t *testing.T) {
	t.Run("To DTO", func(t *testing.T) {
		link := &model.Link{
			ID:         1,
			URLID:      2,
			Href:       "https://example.com",
			IsExternal: true,
			StatusCode: 200,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		dto := link.ToDTO()

		assert.Equal(t, link.ID, dto.ID, "ID should match")
		assert.Equal(t, link.URLID, dto.URLID, "URLID should match")
		assert.Equal(t, link.Href, dto.Href, "Href should match")
		assert.Equal(t, link.IsExternal, dto.IsExternal, "IsExternal should match")
		assert.Equal(t, link.StatusCode, dto.StatusCode, "StatusCode should match")
		assert.WithinDuration(t, link.CreatedAt, dto.CreatedAt, time.Second, "CreatedAt should match")
		assert.WithinDuration(t, link.UpdatedAt, dto.UpdatedAt, time.Second, "UpdatedAt should match")
	})

	t.Run("From Create Input", func(t *testing.T) {
		input := &model.CreateLinkInput{
			URLID:      2,
			Href:       "https://new-example.com",
			IsExternal: true,
			StatusCode: 404,
		}

		link := model.LinkFromCreateInput(input)

		assert.Equal(t, input.URLID, link.URLID, "URLID should match")
		assert.Equal(t, input.Href, link.Href, "Href should match")
		assert.Equal(t, input.IsExternal, link.IsExternal, "IsExternal should match")
		assert.Equal(t, input.StatusCode, link.StatusCode, "StatusCode should match")
		assert.NotZero(t, link.CreatedAt, "CreatedAt should be set")
		assert.NotZero(t, link.UpdatedAt, "UpdatedAt should be set")
	})

	t.Run("Table Name", func(t *testing.T) {
		expected := "links"
		link := model.Link{}

		assert.Equal(t, expected, link.TableName(), "TableName should return 'links'")
	})
}
