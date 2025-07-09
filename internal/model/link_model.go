package model

import (
	"time"

	"gorm.io/gorm"
)

// Link represents a hyperlink found on a URL's page.
type Link struct {
	ID         uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	URLID      uint           `gorm:"not null;index" json:"url_id"`
	Href       string         `gorm:"type:text;not null" json:"href"`
	IsExternal bool           `json:"is_external"`
	StatusCode int            `json:"status_code"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// LinkDTO is a data transfer object for Link responses
type LinkDTO struct {
	ID         uint      `json:"id"`
	URLID      uint      `json:"url_id"`
	Href       string    `json:"href"`
	IsExternal bool      `json:"is_external"`
	StatusCode int       `json:"status_code"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName returns the name of the table for Link.
func (Link) TableName() string {
	return "links"
}

// CreateLinkInput defines fields required to create a new Link.
type CreateLinkInput struct {
	URLID      uint   `json:"url_id" binding:"required"`
	Href       string `json:"href" binding:"required,url"`
	IsExternal bool   `json:"is_external"`
	StatusCode int    `json:"status_code" binding:"required,gte=100,lte=599"`
}

// ToDTO transforms a Link model into a LinkDTO for responses.
func (l *Link) ToDTO() *LinkDTO {
	return &LinkDTO{
		ID:         l.ID,
		URLID:      l.URLID,
		Href:       l.Href,
		IsExternal: l.IsExternal,
		StatusCode: l.StatusCode,
		CreatedAt:  l.CreatedAt,
		UpdatedAt:  l.UpdatedAt,
	}
}

// LinkFromCreateInput maps CreateLinkInput to a Link model instance.
func LinkFromCreateInput(input *CreateLinkInput) *Link {
	now := time.Now()
	return &Link{
		URLID:      input.URLID,
		Href:       input.Href,
		IsExternal: input.IsExternal,
		StatusCode: input.StatusCode,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}
