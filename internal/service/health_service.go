package service

import (
	"time"

	"gorm.io/gorm"
)

type HealthStatus struct {
	Service  string
	Database string
	Healthy  bool
	Checked  time.Time
}
type HealthService interface {
	Check() *HealthStatus
}

type healthService struct {
	db    *gorm.DB
	name  string
	probe func() (string, bool)
}

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
