package model

import (
	"time"

	"gorm.io/gorm"
)

// BlacklistedToken represents an invalidated JWT.
type BlacklistedToken struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"-"`
	JTI       string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"jti"`
	ExpiresAt time.Time      `gorm:"index;not null" json:"expires_at"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BlacklistedTokenDTO is used for API responses (e.g., admin views).
type BlacklistedTokenDTO struct {
	JTI       string    `json:"jti"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName returns the name of the table for BlacklistedToken.
func (BlacklistedToken) TableName() string {
	return "blacklisted_tokens"
}

// CreateBlacklistedTokenInput defines fields to blacklist a token.
type CreateBlacklistedTokenInput struct {
	JTI       string    `json:"jti" binding:"required"`
	ExpiresAt time.Time `json:"expires_at" binding:"required"`
}

// ToDTO converts a BlacklistedToken to its DTO.
func (b *BlacklistedToken) ToDTO() *BlacklistedTokenDTO {
	return &BlacklistedTokenDTO{
		JTI:       b.JTI,
		ExpiresAt: b.ExpiresAt,
		CreatedAt: b.CreatedAt,
	}
}

// FromCreateInput maps CreateBlacklistedTokenInput to a model instance.
func BlacklistedTokenFromCreateInput(input *CreateBlacklistedTokenInput) *BlacklistedToken {
	now := time.Now()
	return &BlacklistedToken{
		JTI:       input.JTI,
		ExpiresAt: input.ExpiresAt,
		CreatedAt: now,
	}
}
