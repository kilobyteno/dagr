// Package email provides pluggable transactional email delivery.
package email

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kilobyteno/dagr-chat/internal/config"
)

// Message is a plain-text transactional email.
type Message struct {
	To      string
	Subject string
	Text    string
}

// Sender delivers transactional email.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// NewSender builds a Sender from host configuration.
// Providers smtp and lettermint fail fast when required settings are missing.
func NewSender(cfg config.Config, logger *slog.Logger) (Sender, error) {
	if logger == nil {
		logger = slog.Default()
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.EmailProvider))
	if provider == "" {
		provider = "log"
	}
	switch provider {
	case "log":
		return NewLogSender(logger), nil
	case "smtp":
		return NewSMTPSender(cfg)
	case "lettermint":
		return NewLettermintSender(cfg)
	default:
		return nil, fmt.Errorf("email: unknown EMAIL_PROVIDER %q (want log, smtp, or lettermint)", cfg.EmailProvider)
	}
}
