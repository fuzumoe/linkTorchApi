package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/fuzumoe/linkTorch-api/internal/service"
)

func TestHealthService(t *testing.T) {
	t.Run("Nil DB", func(t *testing.T) {
		hs := service.NewHealthService(nil, "TestService")
		status := hs.Check()

		if status.Service != "TestService" {
			t.Errorf("expected service 'TestService', got %s", status.Service)
		}
		if status.Database != "disconnected" {
			t.Errorf("expected database 'disconnected', got %s", status.Database)
		}
		if status.Healthy {
			t.Errorf("expected Healthy to be false, got true")
		}
		if time.Since(status.Checked) > time.Minute {
			t.Errorf("unexpected Checked timestamp")
		}
	})

	t.Run("Mock Healthy DB", func(t *testing.T) {
		sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("failed to open sqlmock database: %v", err)
		}
		defer sqlDB.Close()

		mock.ExpectPing().WillReturnError(nil)
		mock.ExpectPing().WillReturnError(nil)

		gdb, err := gorm.Open(mysql.New(mysql.Config{
			Conn:                      sqlDB,
			SkipInitializeWithVersion: true,
		}), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to open gorm db: %v", err)
		}

		hs := service.NewHealthService(gdb, "TestService")
		status := hs.Check()

		if status.Service != "TestService" {
			t.Errorf("expected service 'TestService', got %s", status.Service)
		}
		if status.Database != "healthy" {
			t.Errorf("expected database 'healthy', got %s", status.Database)
		}
		if !status.Healthy {
			t.Errorf("expected Healthy to be true, got false")
		}
		if time.Since(status.Checked) > time.Minute {
			t.Errorf("unexpected Checked timestamp")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled expectations: %v", err)
		}
	})

	t.Run("Mock Un healthy DB", func(t *testing.T) {
		sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		if err != nil {
			t.Fatalf("failed to open sqlmock database: %v", err)
		}
		defer sqlDB.Close()

		mock.ExpectPing().WillReturnError(nil)
		mock.ExpectPing().WillReturnError(errors.New("ping error"))

		gdb, err := gorm.Open(mysql.New(mysql.Config{
			Conn:                      sqlDB,
			SkipInitializeWithVersion: true,
		}), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to open gorm db: %v", err)
		}

		hs := service.NewHealthService(gdb, "TestService")
		status := hs.Check()

		if status.Service != "TestService" {
			t.Errorf("expected service 'TestService', got %s", status.Service)
		}
		if status.Database != "unhealthy" {
			t.Errorf("expected database 'unhealthy', got %s", status.Database)
		}
		if status.Healthy {
			t.Errorf("expected Healthy to be false, got true")
		}
		if time.Since(status.Checked) > time.Minute {
			t.Errorf("unexpected Checked timestamp")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled expectations: %v", err)
		}
	})
}
