package email

import (
	"context"
	"log/slog"
)

// LogSender records emails via structured logging (local development default).
type LogSender struct {
	Logger *slog.Logger
}

// NewLogSender returns a Sender that logs instead of delivering.
func NewLogSender(logger *slog.Logger) *LogSender {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogSender{Logger: logger}
}

// Send logs the outbound message.
func (s *LogSender) Send(_ context.Context, msg Message) error {
	s.Logger.Info("email:send",
		"to", msg.To,
		"subject", msg.Subject,
		"body", msg.Text,
	)
	return nil
}
