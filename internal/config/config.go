package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	AppPort                   string
	GinMode                   string
	DBHost                    string
	DBPort                    string
	DBUser                    string
	DBPassword                string
	DBName                    string
	DBSSLMode                 string
	JWTSecret                 string
	JWTExpiresHours           int
	GoogleClientID            string
	GoogleClientSecret        string
	GoogleRedirectURL         string
	GoogleFrontendRedirectURL string
}

// Load reads .env (if present) and environment variables into a Config struct.
func Load() (*Config, error) {
	_ = godotenv.Load()

	expHours, err := strconv.Atoi(getEnv("JWT_EXPIRES_HOURS", "24"))
	if err != nil || expHours <= 0 {
		expHours = 24
	}

	appPort := getEnv("APP_PORT", "8080")
	defaultGoogleRedirect := fmt.Sprintf("http://localhost:%s/api/v1/auth/google/callback", appPort)

	cfg := &Config{
		AppPort:                   appPort,
		GinMode:                   getEnv("GIN_MODE", "debug"),
		DBHost:                    getEnv("DB_HOST", "localhost"),
		DBPort:                    getEnv("DB_PORT", "5432"),
		DBUser:                    getEnv("DB_USER", "postgres"),
		DBPassword:                getEnv("DB_PASSWORD", "postgres"),
		DBName:                    getEnv("DB_NAME", "ayam_batu"),
		DBSSLMode:                 getEnv("DB_SSLMODE", "disable"),
		JWTSecret:                 getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		JWTExpiresHours:           expHours,
		GoogleClientID:            os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:        os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:         getEnv("GOOGLE_REDIRECT_URL", defaultGoogleRedirect),
		GoogleFrontendRedirectURL: os.Getenv("GOOGLE_FRONTEND_REDIRECT_URL"),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET must not be empty")
	}

	return cfg, nil
}

// DSN builds the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
