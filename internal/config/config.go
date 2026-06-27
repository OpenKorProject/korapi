package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds gateway configuration.
type Config struct {
	// Server
	Port int
	Host string

	// Upstream services
	AuthURL string // korauth endpoint
	VMUrl   string // korvm endpoint
	VolURL  string // korvol endpoint

	// Redis (rate limiting + JWKS cache)
	RedisAddr string

	// JWKS cache
	JWKSCacheTTL time.Duration

	// Rate limiting
	RateLimitRPS   int // requests/sec
	RateLimitBurst int

	// Logging
	LogLevel string // debug, info, warn, error
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Port:           8080,
		Host:           "0.0.0.0",
		AuthURL:        "http://localhost:8081",
		VMUrl:          "http://localhost:8082",
		VolURL:         "http://localhost:8083",
		RedisAddr:      "localhost:6379",
		JWKSCacheTTL:   10 * time.Minute,
		RateLimitRPS:   50,
		RateLimitBurst: 100,
		LogLevel:       "info",
	}

	if p := os.Getenv("HTTP_PORT"); p != "" {
		port, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid HTTP_PORT: %w", err)
		}
		cfg.Port = port
	}

	if h := os.Getenv("KORAPI_HOST"); h != "" {
		cfg.Host = h
	}

	if au := os.Getenv("KORAUTH_URL"); au != "" {
		cfg.AuthURL = strings.TrimRight(au, "/")
	}

	if vu := os.Getenv("KORVM_URL"); vu != "" {
		cfg.VMUrl = strings.TrimRight(vu, "/")
	}

	if voU := os.Getenv("KORVOL_URL"); voU != "" {
		cfg.VolURL = strings.TrimRight(voU, "/")
	}

	if ra := os.Getenv("REDIS_URL"); ra != "" {
		// Strip redis://host:port/db down to host:port
		if strings.HasPrefix(ra, "redis://") {
			ra = strings.TrimPrefix(ra, "redis://")
			if idx := strings.LastIndex(ra, "/"); idx != -1 {
				ra = ra[:idx]
			}
		}
		cfg.RedisAddr = ra
	}

	if ttl := os.Getenv("JWKS_CACHE_TTL"); ttl != "" {
		d, err := time.ParseDuration(ttl)
		if err == nil {
			cfg.JWKSCacheTTL = d
		}
	}

	if rl := os.Getenv("RATE_LIMIT_RPS"); rl != "" {
		if v, err := strconv.Atoi(rl); err == nil {
			cfg.RateLimitRPS = v
		}
	}

	if ll := os.Getenv("LOG_LEVEL"); ll != "" {
		cfg.LogLevel = ll
	}

	return cfg, nil
}
