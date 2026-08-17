package email_test

import (
	"log/slog"
	"testing"

	"github.com/kilobyteno/dagr-chat/internal/config"
	"github.com/kilobyteno/dagr-chat/internal/email"
)

func TestNewSenderLog(t *testing.T) {
	t.Parallel()
	sender, err := email.NewSender(config.Config{EmailProvider: "log"}, slog.Default())
	if err != nil {
		t.Fatalf("new sender: %v", err)
	}
	if _, ok := sender.(*email.LogSender); !ok {
		t.Fatalf("expected LogSender, got %T", sender)
	}
}

func TestNewSenderDefaultLog(t *testing.T) {
	t.Parallel()
	sender, err := email.NewSender(config.Config{}, slog.Default())
	if err != nil {
		t.Fatalf("new sender: %v", err)
	}
	if _, ok := sender.(*email.LogSender); !ok {
		t.Fatalf("expected LogSender, got %T", sender)
	}
}

func TestNewSenderSMTPRequiresSettings(t *testing.T) {
	t.Parallel()
	if _, err := email.NewSender(config.Config{EmailProvider: "smtp"}, slog.Default()); err == nil {
		t.Fatal("expected error for missing SMTP settings")
	}
	if _, err := email.NewSender(config.Config{
		EmailProvider: "smtp",
		EmailFrom:     "noreply@example.com",
	}, slog.Default()); err == nil {
		t.Fatal("expected error for missing SMTP_HOST")
	}
	sender, err := email.NewSender(config.Config{
		EmailProvider: "smtp",
		EmailFrom:     "noreply@example.com",
		SMTPHost:      "smtp.example.com",
		SMTPPort:      587,
		SMTPTLS:       true,
	}, slog.Default())
	if err != nil {
		t.Fatalf("new smtp sender: %v", err)
	}
	if _, ok := sender.(*email.SMTPSender); !ok {
		t.Fatalf("expected SMTPSender, got %T", sender)
	}
}

func TestNewSenderLettermintRequiresSettings(t *testing.T) {
	t.Parallel()
	if _, err := email.NewSender(config.Config{EmailProvider: "lettermint"}, slog.Default()); err == nil {
		t.Fatal("expected error for missing Lettermint settings")
	}
	if _, err := email.NewSender(config.Config{
		EmailProvider: "lettermint",
		EmailFrom:     "noreply@example.com",
	}, slog.Default()); err == nil {
		t.Fatal("expected error for missing LETTERMINT_API_TOKEN")
	}
	sender, err := email.NewSender(config.Config{
		EmailProvider:      "lettermint",
		EmailFrom:          "noreply@example.com",
		LettermintAPIToken: "test-token",
	}, slog.Default())
	if err != nil {
		t.Fatalf("new lettermint sender: %v", err)
	}
	if _, ok := sender.(*email.LettermintSender); !ok {
		t.Fatalf("expected LettermintSender, got %T", sender)
	}
}

func TestNewSenderUnknownProvider(t *testing.T) {
	t.Parallel()
	if _, err := email.NewSender(config.Config{EmailProvider: "ses"}, slog.Default()); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
