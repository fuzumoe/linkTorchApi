package model

import (
	"time"

	"gorm.io/gorm"
)

// URL represents a URL to be analyzed and its processing status.
type URL struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      uint           `gorm:"not null;index" json:"user_id"`
	OriginalURL string         `gorm:"type:text;not null;unique" json:"original_url"`
	Status      string         `gorm:"type:enum('queued','running','done','error');default:'queued';not null" json:"status"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the name of the table for URL.
func (URL) TableName() string {
	return "urls"
}

// URLDTO is the data transfer object for URL.
type URLDTO struct {
	ID          uint      `json:"id"`
	UserID      uint      `json:"user_id"`
	OriginalURL string    `json:"original_url"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateURLInput defines required fields to create a URL.
type CreateURLInput struct {
	UserID      uint   `json:"user_id" binding:"required"`
	OriginalURL string `json:"original_url" binding:"required,url"`
}

// ToDTO converts a URL model to a URLDTO.
func (u *URL) ToDTO() *URLDTO {
	return &URLDTO{
		ID:          u.ID,
		UserID:      u.UserID,
		OriginalURL: u.OriginalURL,
		Status:      u.Status,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// FromCreateInput maps CreateURLInput to a URL model.
func URLFromCreateInput(input *CreateURLInput) *URL {
	now := time.Now()
	return &URL{
		UserID:      input.UserID,
		OriginalURL: input.OriginalURL,
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
