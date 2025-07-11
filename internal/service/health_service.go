package service

import (
	"time"

	"gorm.io/gorm"
)

// HealthStatus represents the outcome of a health check.
type HealthStatus struct {
	Service  string
	Database string
	Healthy  bool
	Checked  time.Time
}

// HealthService defines an interface for checking system health.
type HealthService interface {
	// Check returns the current health status of the application and DB.
	Check() *HealthStatus
}

type healthService struct {
	db    *gorm.DB
	name  string
	probe func() (string, bool)
}

// NewHealthService constructs a HealthService.
func NewHealthService(db *gorm.DB, name string) HealthService {
	return &healthService{
		db:   db,
		name: name,
		probe: func() (string, bool) {
			if db == nil {
				return "disconnected", false
			}
			sqlDB, err := db.DB()
			if err != nil {
				return "unhealthy", false
			}
			if pingErr := sqlDB.Ping(); pingErr != nil {
				return "unhealthy", false
			}
			return "healthy", true
		},
	}
}

func (h *healthService) Check() *HealthStatus {
	dbStatus, ok := h.probe()
	return &HealthStatus{
		Service:  h.name,
		Database: dbStatus,
		Healthy:  ok,
		Checked:  time.Now().UTC(),
	}
}
