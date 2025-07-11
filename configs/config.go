package configs

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the application configuration values.
type Config struct {
	ServerHost          string
	ServerPort          string
	ServerMode          string
	DatabaseHost        string
	DatabasePort        string
	DatabaseUser        string
	DatabasePassword    string
	DatabaseName        string
	DatabaseURL         string
	DevUserEmail        string
	DevUserName         string
	DevUserPassword     string
	LogLevel            string
	JWTSecret           string
	JWTLifetime         time.Duration
	MySQLRootPassword   string
	CORSOrigins         []string
	MaxConcurrentCrawls int
	CrawlTimeout        time.Duration
	UserAgent           string
}

// Load reads configuration exclusively from environment variables (optionally .env file).
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}

	// Server
	cfg.ServerHost = getEnv("HOST", "0.0.0.0")
	cfg.ServerPort = getEnv("PORT", "8080")
	cfg.ServerMode = getEnv("GIN_MODE", "debug")

	// Database
	cfg.DatabaseHost = getEnv("DB_HOST", "localhost")
	cfg.DatabasePort = getEnv("DB_PORT", "3306")
	cfg.DatabaseUser = getEnv("DB_USER", "")
	cfg.DatabasePassword = getEnv("DB_PASSWORD", "")
	cfg.DatabaseName = getEnv("DB_NAME", "")
	cfg.MySQLRootPassword = getEnv("MYSQL_ROOT_PASSWORD", "")
	cfg.DevUserName = getEnv("DEV_USER_NAME", "DevUser")
	cfg.DevUserEmail = getEnv("DEV_USER_EMAIL", "admin@admin.com")
	cfg.DevUserPassword = getEnv("DEV_USER_PASSWORD", "admin123")
	if cfg.DatabaseUser == "" || cfg.DatabasePassword == "" || cfg.DatabaseName == "" {
		return nil, fmt.Errorf("missing required database env vars")
	}
	// Build DSN: user:pass@tcp(host:port)/dbname?parseTime=true
	cfg.DatabaseURL = fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.DatabaseUser, cfg.DatabasePassword,
		cfg.DatabaseHost, cfg.DatabasePort,
		cfg.DatabaseName,
	)

	// Logging & Auth
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")
	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("missing JWT_SECRET environment variable")
	}
	jwtLifetimeStr := getEnv("JWT_LIFETIME", "24h")
	d, err := time.ParseDuration(jwtLifetimeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_LIFETIME: %w", err)
	}
	cfg.JWTLifetime = d

	// CORS
	origins := getEnv("CORS_ORIGINS", "")
	if origins != "" {
		cfg.CORSOrigins = strings.Split(origins, ",")
	}

	// Crawling
	maxCrawls := getEnv("MAX_CONCURRENT_CRAWLS", "5")
	mc, err := strconv.Atoi(maxCrawls)
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_CONCURRENT_CRAWLS: %w", err)
	}
	cfg.MaxConcurrentCrawls = mc

	timeoutSec := getEnv("CRAWL_TIMEOUT_SECONDS", "30")
	ts, err := strconv.Atoi(timeoutSec)
	if err != nil {
		return nil, fmt.Errorf("invalid CRAWL_TIMEOUT_SECONDS: %w", err)
	}
	cfg.CrawlTimeout = time.Duration(ts) * time.Second

	// User agent
	cfg.UserAgent = getEnv("USER_AGENT", "URLInsight-Bot/1.0")

	return cfg, nil
}

// getEnv returns env var or default.
func getEnv(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}
