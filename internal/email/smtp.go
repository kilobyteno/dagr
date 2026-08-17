package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/kilobyteno/dagr-chat/internal/config"
)

// SMTPSender delivers mail over SMTP.
type SMTPSender struct {
	host     string
	port     int
	username string
	password string
	from     string
	useTLS   bool
}

// NewSMTPSender validates SMTP settings and returns a Sender.
func NewSMTPSender(cfg config.Config) (*SMTPSender, error) {
	from := strings.TrimSpace(cfg.EmailFrom)
	host := strings.TrimSpace(cfg.SMTPHost)
	if from == "" {
		return nil, fmt.Errorf("email: EMAIL_FROM is required for EMAIL_PROVIDER=smtp")
	}
	if host == "" {
		return nil, fmt.Errorf("email: SMTP_HOST is required for EMAIL_PROVIDER=smtp")
	}
	port := cfg.SMTPPort
	if port <= 0 {
		port = 587
	}
	return &SMTPSender{
		host:     host,
		port:     port,
		username: cfg.SMTPUsername,
		password: cfg.SMTPPassword,
		from:     from,
		useTLS:   cfg.SMTPTLS,
	}, nil
}

// Send delivers a plain-text message via SMTP.
func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	to := strings.TrimSpace(msg.To)
	if to == "" {
		return fmt.Errorf("email: missing recipient")
	}
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	body := buildPlainMessage(s.from, to, msg.Subject, msg.Text)

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("email: smtp dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("email: smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if s.useTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: s.host}); err != nil {
				return fmt.Errorf("email: smtp starttls: %w", err)
			}
		}
	}

	if s.username != "" {
		auth := smtp.PlainAuth("", s.username, s.password, s.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("email: smtp auth: %w", err)
		}
	}

	if err := client.Mail(extractAddress(s.from)); err != nil {
		return fmt.Errorf("email: smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("email: smtp rcpt: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("email: smtp data: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		_ = w.Close()
		return fmt.Errorf("email: smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("email: smtp close data: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("email: smtp quit: %w", err)
	}
	return nil
}

func buildPlainMessage(from, to, subject, text string) []byte {
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(from)
	b.WriteString("\r\n")
	b.WriteString("To: ")
	b.WriteString(to)
	b.WriteString("\r\n")
	b.WriteString("Subject: ")
	b.WriteString(sanitizeHeader(subject))
	b.WriteString("\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(text)
	if !strings.HasSuffix(text, "\n") {
		b.WriteString("\r\n")
	}
	return []byte(b.String())
}

func sanitizeHeader(v string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, v)
}

func extractAddress(from string) string {
	from = strings.TrimSpace(from)
	if i := strings.LastIndex(from, "<"); i >= 0 {
		if j := strings.Index(from[i:], ">"); j > 0 {
			return strings.TrimSpace(from[i+1 : i+j])
		}
	}
	return from
}
