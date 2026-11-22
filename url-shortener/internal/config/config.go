package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// defaultJWTSecret is the fallback secret used outside production so that a
// fresh checkout works out of the box. Production never accepts it.
const defaultJWTSecret = "default-secret-change-in-production"

// minJWTSecretLen is the minimum accepted JWT secret length outside
// development.
const minJWTSecretLen = 32

// knownWeakSecrets are publicly known values that must never be used as the
// JWT secret outside development: the in-code fallback and the value that
// used to ship in .env.example.
var knownWeakSecrets = map[string]bool{
	defaultJWTSecret: true,
	"your-super-secret-key-change-in-production": true,
}

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
	// TrustedProxies lists proxy IPs/CIDRs whose X-Forwarded-For header is
	// trusted when resolving the client IP. Empty trusts nobody.
	TrustedProxies []string
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

	trustedProxies, err := getTrustedProxies()
	if err != nil {
		return nil, err
	}
	jwtExpiry, err := getDuration("JWT_EXPIRY", 24*time.Hour)
	if err != nil {
		return nil, err
	}
	globalPerMinute, err := getInt("RATE_LIMIT_GLOBAL", 300)
	if err != nil {
		return nil, err
	}
	authPerMinute, err := getInt("RATE_LIMIT_AUTH", 10)
	if err != nil {
		return nil, err
	}

	config := &Config{
		App: AppConfig{
			Env:            env,
			Port:           port,
			BaseURL:        baseURL,
			AllowedOrigins: getAllowedOrigins(env, baseURL),
			TrustedProxies: trustedProxies,
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
			Expiry: jwtExpiry,
		},
		RateLimit: RateLimitConfig{
			GlobalPerMinute: globalPerMinute,
			AuthPerMinute:   authPerMinute,
		},
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// validate fails fast on unsafe configuration in every environment except
// development, instead of booting with insecure defaults
func (c *Config) validate() error {
	if c.IsDevelopment() {
		return nil
	}
	if knownWeakSecrets[c.JWT.Secret] {
		return errors.New("JWT_SECRET is set to a publicly known value; generate a fresh one with: openssl rand -base64 48")
	}
	if len(c.JWT.Secret) < minJWTSecretLen {
		return fmt.Errorf("JWT_SECRET must be at least %d characters unless APP_ENV=development", minJWTSecretLen)
	}
	if c.Database.Password == "" || c.Database.Password == "postgres" {
		return errors.New("DB_PASSWORD must be set to a strong value unless APP_ENV=development")
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

// getInt gets an environment variable as a positive integer; unset falls back
// to the default, anything else invalid is an error
func getInt(key string, defaultValue int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s=%q must be a positive integer", key, value)
	}
	return n, nil
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

// getDuration parses a duration environment variable; unset falls back to the
// default, garbage and non-positive values are errors
func getDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s=%q is not a valid duration (e.g. 24h, 30m)", key, value)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s=%q must be a positive duration", key, value)
	}
	return d, nil
}

// getTrustedProxies parses TRUSTED_PROXIES (comma-separated IPs or CIDRs).
// An empty value trusts nobody: the client IP is always the direct peer.
func getTrustedProxies() ([]string, error) {
	raw := os.Getenv("TRUSTED_PROXIES")
	if raw == "" {
		return nil, nil
	}
	var proxies []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(p); err != nil && net.ParseIP(p) == nil {
			return nil, fmt.Errorf("TRUSTED_PROXIES: %q is not a valid IP or CIDR", p)
		}
		proxies = append(proxies, p)
	}
	return proxies, nil
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
