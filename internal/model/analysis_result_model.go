package model

import (
	"time"

	"gorm.io/gorm"
)

// AnalysisResult holds parsed metadata for a given URL.
type AnalysisResult struct {
	ID                uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	URLID             uint           `gorm:"not null;index" json:"url_id"`
	HTMLVersion       string         `gorm:"size:50;not null" json:"html_version"`
	Title             string         `gorm:"type:text" json:"title"`
	H1Count           int            `json:"h1_count"`
	H2Count           int            `json:"h2_count"`
	H3Count           int            `json:"h3_count"`
	H4Count           int            `json:"h4_count"`
	H5Count           int            `json:"h5_count"`
	H6Count           int            `json:"h6_count"`
	HasLoginForm      bool           `json:"has_login_form"`
	InternalLinkCount int            `json:"internal_link_count"`
	ExternalLinkCount int            `json:"external_link_count"`
	BrokenLinkCount   int            `json:"broken_link_count"`
	CreatedAt         time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// AnalysisResultDTO is used for sending analysis results in responses.
type AnalysisResultDTO struct {
	ID           uint      `json:"id"`
	URLID        uint      `json:"url_id"`
	HTMLVersion  string    `json:"html_version"`
	Title        string    `json:"title"`
	H1Count      int       `json:"h1_count"`
	H2Count      int       `json:"h2_count"`
	H3Count      int       `json:"h3_count"`
	H4Count      int       `json:"h4_count"`
	H5Count      int       `json:"h5_count"`
	H6Count      int       `json:"h6_count"`
	HasLoginForm bool      `json:"has_login_form"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName returns the name of the table for AnalysisResult.
func (AnalysisResult) TableName() string {
	return "analysis_results"
}

// CreateAnalysisResultInput defines required fields to create an analysis result.
type CreateAnalysisResultInput struct {
	URLID        uint   `json:"url_id" binding:"required"`
	HTMLVersion  string `json:"html_version" binding:"required"`
	Title        string `json:"title" binding:"omitempty"`
	H1Count      int    `json:"h1_count" binding:"gte=0"`
	H2Count      int    `json:"h2_count" binding:"gte=0"`
	H3Count      int    `json:"h3_count" binding:"gte=0"`
	H4Count      int    `json:"h4_count" binding:"gte=0"`
	H5Count      int    `json:"h5_count" binding:"gte=0"`
	H6Count      int    `json:"h6_count" binding:"gte=0"`
	HasLoginForm bool   `json:"has_login_form"`
}

// ToDTO converts an AnalysisResult model to AnalysisResultDTO.
func (r *AnalysisResult) ToDTO() *AnalysisResultDTO {
	return &AnalysisResultDTO{
		ID:           r.ID,
		URLID:        r.URLID,
		HTMLVersion:  r.HTMLVersion,
		Title:        r.Title,
		H1Count:      r.H1Count,
		H2Count:      r.H2Count,
		H3Count:      r.H3Count,
		H4Count:      r.H4Count,
		H5Count:      r.H5Count,
		H6Count:      r.H6Count,
		HasLoginForm: r.HasLoginForm,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

// FromCreateInput maps CreateAnalysisResultInput to AnalysisResult model.
func AnalysisResultFromCreateInput(input *CreateAnalysisResultInput) *AnalysisResult {
	now := time.Now()
	return &AnalysisResult{
		URLID:        input.URLID,
		HTMLVersion:  input.HTMLVersion,
		Title:        input.Title,
		H1Count:      input.H1Count,
		H2Count:      input.H2Count,
		H3Count:      input.H3Count,
		H4Count:      input.H4Count,
		H5Count:      input.H5Count,
		H6Count:      input.H6Count,
		HasLoginForm: input.HasLoginForm,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
