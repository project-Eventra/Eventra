package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	Port            string
	Environment     string
	DatabaseURL     string
	JWTSecret       string
	JWTIssuer       string
	JWTAudience     string
	FirebaseProject string
	FrontendOrigin  string
	CookieDomain    string
	CookieSecure    bool
	RateLimitRPS    int
	RateLimitBurst  int
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:            getEnv("PORT", "8080"),
		Environment:     getEnv("ENVIRONMENT", "development"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		JWTIssuer:       getEnv("JWT_ISSUER", "evenrtra"),
		JWTAudience:     getEnv("JWT_AUDIENCE", "evenrtra-web"),
		FirebaseProject: os.Getenv("FIREBASE_PROJECT_ID"),
		FrontendOrigin:  getEnv("FRONTEND_ORIGIN", "http://localhost:5500"),
		CookieDomain:    getEnv("COOKIE_DOMAIN", "localhost"),
		CookieSecure:    getEnvBool("COOKIE_SECURE", false),
		RateLimitRPS:    getEnvInt("RATE_LIMIT_RPS", 10),
		RateLimitBurst:  getEnvInt("RATE_LIMIT_BURST", 20),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required (minimum 32 characters)")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	return cfg, nil
}

// IsProd returns true if the environment is production
func (c *Config) IsProd() bool {
	return c.Environment == "production"
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		b, err := strconv.ParseBool(val)
		if err != nil {
			return fallback
		}
		return b
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		i, err := strconv.Atoi(val)
		if err != nil {
			return fallback
		}
		return i
	}
	return fallback
}
