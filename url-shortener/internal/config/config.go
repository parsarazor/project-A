package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// defaultJWTSecret is the fallback secret used outside production so that a
// fresh checkout works out of the box. Production never accepts it.
const defaultJWTSecret = "default-secret-change-in-production"

// minJWTSecretLen is the minimum accepted JWT secret length in production.
const minJWTSecretLen = 32

// Config holds all configuration for the application
type Config struct {
	App       AppConfig
	Database  DatabaseConfig
	JWT       JWTConfig
	RateLimit RateLimitConfig
}

// AppConfig holds application-specific configuration
type AppConfig struct {
	Env            string
	Port           string
	BaseURL        string
	AllowedOrigins []string
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	Secret string
	Expiry time.Duration
}

// RateLimitConfig holds per-IP rate limiting settings (requests per minute)
type RateLimitConfig struct {
	GlobalPerMinute int
	AuthPerMinute   int
}

// Load loads configuration from environment variables and validates it
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	env := getEnv("APP_ENV", "development")
	port := getEnv("APP_PORT", "8080")
	baseURL := getEnv("APP_BASE_URL", "http://localhost:8080")

	config := &Config{
		App: AppConfig{
			Env:            env,
			Port:           port,
			BaseURL:        baseURL,
			AllowedOrigins: getAllowedOrigins(env, baseURL),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "urlshortener"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", defaultJWTSecret),
			Expiry: parseDuration(getEnv("JWT_EXPIRY", "24h")),
		},
		RateLimit: RateLimitConfig{
			GlobalPerMinute: getInt("RATE_LIMIT_GLOBAL", 300),
			AuthPerMinute:   getInt("RATE_LIMIT_AUTH", 10),
		},
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// validate fails fast on unsafe production configuration instead of booting
// with insecure defaults
func (c *Config) validate() error {
	if c.IsProduction() {
		if c.JWT.Secret == defaultJWTSecret {
			return errors.New("JWT_SECRET must be set in production; generate one with: openssl rand -base64 48")
		}
		if len(c.JWT.Secret) < minJWTSecretLen {
			return fmt.Errorf("JWT_SECRET must be at least %d characters in production", minJWTSecretLen)
		}
		if c.Database.Password == "" || c.Database.Password == "postgres" {
			return errors.New("DB_PASSWORD must be set to a strong value in production")
		}
	}
	return nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getInt gets an environment variable as an integer or returns a default value
func getInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var n int
		if _, err := fmt.Sscanf(value, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return defaultValue
}

// getAllowedOrigins parses ALLOWED_ORIGINS (comma-separated). It defaults to
// "*" in development and to the API's own base URL origin otherwise.
func getAllowedOrigins(env, baseURL string) []string {
	if raw := os.Getenv("ALLOWED_ORIGINS"); raw != "" {
		var origins []string
		for _, o := range strings.Split(raw, ",") {
			if o = strings.TrimSpace(o); o != "" {
				origins = append(origins, o)
			}
		}
		if len(origins) > 0 {
			return origins
		}
	}
	if env == "development" {
		return []string{"*"}
	}
	return []string{baseURL}
}

// parseDuration parses a duration string, returns default on error
func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}

// IsDevelopment returns true if the app is running in development mode
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

// IsProduction returns true if the app is running in production mode
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

// IsStaging returns true if the app is running in staging mode
func (c *Config) IsStaging() bool {
	return c.App.Env == "staging"
}
