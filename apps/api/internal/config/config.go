package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env               string
	HTTPAddr          string
	DatabaseURL       string
	AppOrigin         string
	CookieSecure      bool
	SessionTTL        time.Duration
	TokenSecret       string
	RegisterRateLimit int
	SMTP              SMTP
}

type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      bool
	StartTLS bool
}

func Load() (Config, error) {
	loadDotEnv("apps/api/.env")
	loadDotEnv(".env")

	registerLimit, err := strconv.Atoi(env("REGISTER_RATE_LIMIT", "10"))
	if err != nil || registerLimit < 1 {
		return Config{}, fmt.Errorf("invalid REGISTER_RATE_LIMIT")
	}
	port, err := strconv.Atoi(env("SMTP_PORT", "1025"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}
	ttl, err := time.ParseDuration(env("SESSION_TTL", "336h"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid SESSION_TTL: %w", err)
	}
	secure, err := strconv.ParseBool(env("COOKIE_SECURE", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid COOKIE_SECURE: %w", err)
	}
	tlsEnabled, err := strconv.ParseBool(env("SMTP_TLS", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid SMTP_TLS: %w", err)
	}
	startTLS, err := strconv.ParseBool(env("SMTP_STARTTLS", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid SMTP_STARTTLS: %w", err)
	}

	cfg := Config{
		Env:               env("APP_ENV", "development"),
		HTTPAddr:          env("HTTP_ADDR", ":8000"),
		DatabaseURL:       env("DATABASE_URL", "postgres://postgres:postgres@localhost:55432/cestia?sslmode=disable"),
		AppOrigin:         strings.TrimRight(env("APP_ORIGIN", "http://localhost:5173"), "/"),
		CookieSecure:      secure,
		SessionTTL:        ttl,
		TokenSecret:       env("TOKEN_SECRET", "development-only-secret-change-before-production"),
		RegisterRateLimit: registerLimit,
		SMTP: SMTP{
			Host:     env("SMTP_HOST", "localhost"),
			Port:     port,
			Username: env("SMTP_USERNAME", ""),
			Password: env("SMTP_PASSWORD", ""),
			From:     env("SMTP_FROM", "Cestia <no-reply@cestia.local>"),
			TLS:      tlsEnabled,
			StartTLS: startTLS,
		},
	}
	if cfg.Env == "production" {
		if len(cfg.TokenSecret) < 32 || strings.Contains(cfg.TokenSecret, "development-only") {
			return Config{}, fmt.Errorf("TOKEN_SECRET must be a production secret of at least 32 characters")
		}
		if !cfg.CookieSecure {
			return Config{}, fmt.Errorf("COOKIE_SECURE must be true in production")
		}
		if !strings.HasPrefix(cfg.AppOrigin, "https://") {
			return Config{}, fmt.Errorf("APP_ORIGIN must use https in production")
		}
		if cfg.SMTP.StartTLS == false && cfg.SMTP.TLS == false {
			return Config{}, fmt.Errorf("SMTP must use TLS or STARTTLS in production")
		}
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(strings.TrimSpace(key)); !exists {
			_ = os.Setenv(strings.TrimSpace(key), strings.Trim(strings.TrimSpace(value), `"'`))
		}
	}
}
