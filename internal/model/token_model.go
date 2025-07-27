// internal/model/token_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TokenClaims embeds jwt.StandardClaims and adds a UserID field.

// NewJTI generates a new unique identifier for the JWT ID (jti) claim.
func NewJTI() string {
	return uuid.NewString()
}

// BlacklistedToken represents a revoked JWT ID (jti) with its expiration.
type BlacklistedToken struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"-"`
	JTI       string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"jti"`
	ExpiresAt time.Time      `gorm:"index;not null" json:"expires_at"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides GORM’s default table name.
func (BlacklistedToken) TableName() string {
	return "blacklisted_tokens"
}

// BlacklistedTokenDTO is the data transfer object for a revoked token.
type BlacklistedTokenDTO struct {
	JTI       string    `json:"jti"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ToDTO converts a BlacklistedToken to its DTO form.
func (b *BlacklistedToken) ToDTO() *BlacklistedTokenDTO {
	return &BlacklistedTokenDTO{
		JTI:       b.JTI,
		ExpiresAt: b.ExpiresAt,
	}
}

// FromJTI constructs a BlacklistedToken from a jti string and expiration time.
func FromJTI(jti string, exp time.Time) *BlacklistedToken {
	return &BlacklistedToken{
		JTI:       jti,
		ExpiresAt: exp,
	}
}
