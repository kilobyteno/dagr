// Package config loads runtime configuration from the environment.
package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kilobyteno/dagr-chat/internal/auth"
)

// DeploymentMode selects self-hosted (unlimited, no billing) or Cloud (plans enabled).
type DeploymentMode string

const (
	DeploymentSelfHosted DeploymentMode = "selfhosted"
	DeploymentCloud      DeploymentMode = "cloud"
)

// Config holds process configuration for the Dagr server and workers.
type Config struct {
	HTTPAddr                string
	PublicBaseURL           string
	DatabaseURL             string
	RedisURL                string
	RedisTLS                bool
	RedisTLSSkipVerify      bool
	S3Endpoint              string
	S3AccessKey             string
	S3SecretKey             string
	S3Bucket                string
	S3UseSSL                bool
	S3Region                string
	PasswordPolicy          auth.PasswordPolicy
	SessionTTL              time.Duration
	ServerID                string
	ServerPublicURL         string
	ServerSigningPrivateKey string
	EmailProvider           string
	EmailFrom               string
	SMTPHost                string
	SMTPPort                int
	SMTPUsername            string
	SMTPPassword            string
	SMTPTLS                 bool
	LettermintAPIToken      string
	LettermintBaseURL       string
	DeploymentMode          DeploymentMode
	BillingCurrency         string
	ProMonthlyCents         int
	YearlyDiscountPercent   int
	EarlyAccessEnabled      bool
	EarlyAccessMonths       int
	EarlyAccessPercentOff   int
	BillingGracePeriod      time.Duration
	MollieAPIKey            string
	MollieProfileID         string
	MollieWebhookURL        string
	LogLevel                slog.Level
}

// DefaultPasswordMinLength is used when PASSWORD_MIN_LENGTH is unset or invalid.
const DefaultPasswordMinLength = 12

// DefaultSessionTTL is used when SESSION_TTL is unset or invalid.
const DefaultSessionTTL = 30 * 24 * time.Hour

// Load reads configuration from environment variables with local-development defaults.
func Load() Config {
	return Config{
		HTTPAddr:      getenv("HTTP_ADDR", ":8080"),
		PublicBaseURL: getenv("PUBLIC_BASE_URL", "http://localhost:5173"),
		DatabaseURL:   getenv("DATABASE_URL", "postgres://dagr:dagr@localhost:5433/dagr?sslmode=disable"),
		RedisURL:           getenv("REDIS_URL", "redis://localhost:6379/0"),
		RedisTLS:           getenvBool("REDIS_TLS", false),
		RedisTLSSkipVerify: getenvBool("REDIS_TLS_SKIP_VERIFY", false),
		S3Endpoint:    getenv("S3_ENDPOINT", "localhost:9000"),
		S3AccessKey:   getenv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:   getenv("S3_SECRET_KEY", "minioadmin"),
		S3Bucket:      getenv("S3_BUCKET", "dagr"),
		S3UseSSL:      getenvBool("S3_USE_SSL", false),
		S3Region:      getenv("S3_REGION", "us-east-1"),
		PasswordPolicy: auth.PasswordPolicy{
			MinLength:        getenvInt("PASSWORD_MIN_LENGTH", DefaultPasswordMinLength, 1),
			RequireUppercase: getenvBool("PASSWORD_REQUIRE_UPPERCASE", true),
			RequireLowercase: getenvBool("PASSWORD_REQUIRE_LOWERCASE", true),
			RequireNumber:    getenvBool("PASSWORD_REQUIRE_NUMBER", true),
			RequireSymbol:    getenvBool("PASSWORD_REQUIRE_SYMBOL", false),
		},
		SessionTTL:              getenvDuration("SESSION_TTL", DefaultSessionTTL),
		ServerID:                getenv("SERVER_ID", ""),
		ServerPublicURL:         getenv("SERVER_PUBLIC_URL", ""),
		ServerSigningPrivateKey: getenv("SERVER_SIGNING_PRIVATE_KEY", ""),
		EmailProvider:           strings.ToLower(getenv("EMAIL_PROVIDER", "log")),
		EmailFrom:               getenv("EMAIL_FROM", ""),
		SMTPHost:                getenv("SMTP_HOST", ""),
		SMTPPort:                getenvInt("SMTP_PORT", 587, 1),
		SMTPUsername:            getenv("SMTP_USERNAME", ""),
		SMTPPassword:            getenv("SMTP_PASSWORD", ""),
		SMTPTLS:                 getenvBool("SMTP_TLS", true),
		LettermintAPIToken:      getenv("LETTERMINT_API_TOKEN", ""),
		LettermintBaseURL:       getenv("LETTERMINT_BASE_URL", ""),
		DeploymentMode:          parseDeploymentMode(getenv("DEPLOYMENT_MODE", string(DeploymentSelfHosted))),
		BillingCurrency:         strings.ToUpper(getenv("BILLING_CURRENCY", "EUR")),
		ProMonthlyCents:         getenvInt("PRO_MONTHLY_CENTS", 700, 1),
		YearlyDiscountPercent:   getenvInt("YEARLY_DISCOUNT_PERCENT", 10, 0),
		EarlyAccessEnabled:      getenvBool("EARLY_ACCESS_ENABLED", true),
		EarlyAccessMonths:       getenvInt("EARLY_ACCESS_MONTHS", 3, 1),
		EarlyAccessPercentOff:   getenvInt("EARLY_ACCESS_PERCENT_OFF", 50, 0),
		BillingGracePeriod:      getenvDuration("BILLING_GRACE_PERIOD", 7*24*time.Hour),
		MollieAPIKey:            getenv("MOLLIE_API_KEY", ""),
		MollieProfileID:         getenv("MOLLIE_PROFILE_ID", ""),
		MollieWebhookURL:        getenv("MOLLIE_WEBHOOK_URL", ""),
		LogLevel:                parseLogLevel(getenv("LOG_LEVEL", "info")),
	}
}

// NewLogger returns a JSON slog logger at c.LogLevel and sets it as the process default.
func (c Config) NewLogger() *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: c.LogLevel}))
	slog.SetDefault(logger)
	return logger
}

func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// IsCloud reports whether this process is the hosted Cloud deployment.
func (c Config) IsCloud() bool {
	return c.DeploymentMode == DeploymentCloud
}

// BillingEnabled is true only on Cloud. Self-hosted never charges.
func (c Config) BillingEnabled() bool {
	return c.IsCloud()
}

func parseDeploymentMode(value string) DeploymentMode {
	switch DeploymentMode(strings.ToLower(strings.TrimSpace(value))) {
	case DeploymentCloud:
		return DeploymentCloud
	default:
		return DeploymentSelfHosted
	}
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getenvInt(key string, fallback, min int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < min {
		return fallback
	}
	return n
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
