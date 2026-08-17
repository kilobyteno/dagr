package email

import (
	"context"
	"fmt"
	"strings"

	lettermint "github.com/lettermint/lettermint-go"

	"github.com/kilobyteno/dagr-chat/internal/config"
)

// LettermintSender delivers mail via the Lettermint API.
type LettermintSender struct {
	client *lettermint.Client
	from   string
}

// NewLettermintSender validates Lettermint settings and returns a Sender.
func NewLettermintSender(cfg config.Config) (*LettermintSender, error) {
	from := strings.TrimSpace(cfg.EmailFrom)
	token := strings.TrimSpace(cfg.LettermintAPIToken)
	if from == "" {
		return nil, fmt.Errorf("email: EMAIL_FROM is required for EMAIL_PROVIDER=lettermint")
	}
	if token == "" {
		return nil, fmt.Errorf("email: LETTERMINT_API_TOKEN is required for EMAIL_PROVIDER=lettermint")
	}
	opts := []lettermint.Option{}
	if base := strings.TrimSpace(cfg.LettermintBaseURL); base != "" {
		opts = append(opts, lettermint.WithBaseURL(base))
	}
	client, err := lettermint.New(token, opts...)
	if err != nil {
		return nil, fmt.Errorf("email: lettermint client: %w", err)
	}
	return &LettermintSender{client: client, from: from}, nil
}

// Send delivers a plain-text message via Lettermint.
func (s *LettermintSender) Send(ctx context.Context, msg Message) error {
	to := strings.TrimSpace(msg.To)
	if to == "" {
		return fmt.Errorf("email: missing recipient")
	}
	_, err := s.client.Email(ctx).
		From(s.from).
		To(to).
		Subject(msg.Subject).
		Text(msg.Text).
		Send()
	if err != nil {
		return fmt.Errorf("email: lettermint send: %w", err)
	}
	return nil
}
