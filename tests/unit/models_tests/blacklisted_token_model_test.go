package model_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/urlinsight-backend/configs"
)

func TestLoad(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Set required env vars
		os.Setenv("DB_USER", "user")
		os.Setenv("DB_PASSWORD", "pass")
		os.Setenv("DB_NAME", "db")
		os.Setenv("JWT_SECRET", "secret")

		// Optional overrides
		os.Setenv("HOST", "127.0.0.1")
		os.Setenv("PORT", "9090")
		os.Setenv("GIN_MODE", "release")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("JWT_LIFETIME", "48h")
		os.Setenv("CORS_ORIGINS", "http://a.com,http://b.com")
		os.Setenv("MAX_CONCURRENT_CRAWLS", "10")
		os.Setenv("CRAWL_TIMEOUT_SECONDS", "45")
		os.Setenv("USER_AGENT", "TestAgent/2.0")

		cfg, err := configs.Load()
		assert.NoError(t, err)
		assert.Equal(t, "127.0.0.1", cfg.ServerHost)
		assert.Equal(t, "9090", cfg.ServerPort)
		assert.Equal(t, "release", cfg.ServerMode)
		assert.Equal(t, []string{"http://a.com", "http://b.com"}, cfg.CORSOrigins)
		assert.Equal(t, 10, cfg.MaxConcurrentCrawls)
		assert.Equal(t, 45*time.Second, cfg.CrawlTimeout)
		assert.Equal(t, "TestAgent/2.0", cfg.UserAgent)
		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Equal(t, "secret", cfg.JWTSecret)
		assert.Equal(t, 48*time.Hour, cfg.JWTLifetime)

		expectedDSN := "user:pass@tcp(localhost:3306)/db?parseTime=true"
		assert.Equal(t, expectedDSN, cfg.DatabaseURL)
	})

	t.Run("Missing DB Env", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("JWT_SECRET", "s")
		_, err := configs.Load()
		assert.EqualError(t, err, "missing required database env vars")
	})

	t.Run("Missing JWT Secret", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("DB_USER", "u")
		os.Setenv("DB_PASSWORD", "p")
		os.Setenv("DB_NAME", "n")
		_, err := configs.Load()
		assert.EqualError(t, err, "missing JWT_SECRET environment variable")
	})

	t.Run("Invalid JWT Lifetime", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("DB_USER", "u")
		os.Setenv("DB_PASSWORD", "p")
		os.Setenv("DB_NAME", "n")
		os.Setenv("JWT_SECRET", "s")
		os.Setenv("JWT_LIFETIME", "invalid")
		_, err := configs.Load()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JWT_LIFETIME")
	})
}
