package model_test

import (
	"testing"
	"time"

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

	if dto.ID != url.ID {
		t.Errorf("ToDTO ID = %d; want %d", dto.ID, url.ID)
	}
	if dto.UserID != url.UserID {
		t.Errorf("ToDTO UserID = %d; want %d", dto.UserID, url.UserID)
	}
	if dto.OriginalURL != url.OriginalURL {
		t.Errorf("ToDTO OriginalURL = %s; want %s", dto.OriginalURL, url.OriginalURL)
	}
	if dto.Status != url.Status {
		t.Errorf("ToDTO Status = %s; want %s", dto.Status, url.Status)
	}
	if !dto.CreatedAt.Equal(url.CreatedAt) {
		t.Errorf("ToDTO CreatedAt = %v; want %v", dto.CreatedAt, url.CreatedAt)
	}
	if !dto.UpdatedAt.Equal(url.UpdatedAt) {
		t.Errorf("ToDTO UpdatedAt = %v; want %v", dto.UpdatedAt, url.UpdatedAt)
	}
}

// TestURLFromCreateInput tests the conversion from CreateURLInput to URL model.
func TestURLFromCreateInput(t *testing.T) {
	input := &model.CreateURLInput{
		UserID:      2,
		OriginalURL: "https://new-example.com",
	}

	url := model.URLFromCreateInput(input)

	if url.UserID != input.UserID {
		t.Errorf("URLFromCreateInput UserID = %d; want %d", url.UserID, input.UserID)
	}
	if url.OriginalURL != input.OriginalURL {
		t.Errorf("URLFromCreateInput OriginalURL = %s; want %s", url.OriginalURL, input.OriginalURL)
	}
	if url.Status != "queued" {
		t.Errorf("URLFromCreateInput Status = %s; want 'queued'", url.Status)
	}
}

// TestURLTableName tests the TableName method of the URL model.
func TestURLTableName(t *testing.T) {
	expected := "urls"
	url := model.URL{}
	// Check if the TableName method returns the expected table name
	if url.TableName() != expected {
		t.Errorf("TableName = %s; want %s", url.TableName(), expected)
	}
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

	if dto.ID != 1 {
		t.Errorf("URLDTO ID = %d; want 1", dto.ID)
	}
	if dto.UserID != 2 {
		t.Errorf("URLDTO UserID = %d; want 2", dto.UserID)
	}
	if dto.OriginalURL != "https://example.com" {
		t.Errorf("URLDTO OriginalURL = %s; want 'https://example.com'", dto.OriginalURL)
	}
	if dto.Status != "done" {
		t.Errorf("URLDTO Status = %s; want 'done'", dto.Status)
	}
	if !dto.CreatedAt.Equal(createdAt) {
		t.Errorf("URLDTO CreatedAt = %v; want %v", dto.CreatedAt, createdAt)
	}
	if !dto.UpdatedAt.Equal(updatedAt) {
		t.Errorf("URLDTO UpdatedAt = %v; want %v", dto.UpdatedAt, updatedAt)
	}
}
