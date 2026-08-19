// Package config loads runtime settings from environment variables.
package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string

	// AppBaseURL is the public URL of the frontend, used to build links in
	// transactional email (e.g. the verify-email link) and billing redirects.
	AppBaseURL string

	// --- Resend (transactional email) ---
	ResendAPIKey  string
	EmailFromAddr string // e.g. "SS Tax Engine <onboarding@resend.dev>"

	// --- Dodo Payments (billing) ---
	DodoAPIKey        string
	DodoWebhookSecret string
	DodoProProductID  string
	DodoEnv           string // "test_mode" | "live_mode"
}

// Load reads configuration from the environment, applying sensible defaults for
// local development and failing only when a security-critical value is missing.
// Email/billing keys are intentionally NOT required: when they are absent the
// app runs in a dev-friendly degraded mode (verification links are logged to
// the console; billing checkout returns a clear "not configured" error) so the
// whole stack still starts with a single `make up`.
func Load() (*Config, error) {
	c := &Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),

		AppBaseURL: getenv("APP_BASE_URL", "http://localhost:5174"),

		// Secret-type keys are cleaned of stray whitespace and surrounding
		// quotes so a copy/paste slip in .env (e.g. `KEY= "abc"`) doesn't turn
		// into a silently invalid credential.
		ResendAPIKey:  secret("RESEND_API_KEY"),
		EmailFromAddr: getenv("EMAIL_FROM", "SS Tax Engine <onboarding@resend.dev>"),

		DodoAPIKey:        secret("DODO_PAYMENTS_API_KEY"),
		DodoWebhookSecret: secret("DODO_PAYMENTS_WEBHOOK_SECRET"),
		DodoProProductID:  secret("DODO_PRO_PRODUCT_ID"),
		DodoEnv:           getenv("DODO_ENV", "test_mode"),
	}
	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	return c, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// secret reads an env var and strips surrounding whitespace and a single layer
// of matching quotes — guarding against common .env paste mistakes.
func secret(k string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
	}
	return strings.TrimSpace(v)
}
