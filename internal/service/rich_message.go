package service

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kilobyteno/dagr-chat/internal/domain"
)

func ParseIncomingWebhookPayload(raw []byte) (domain.RichPayload, error) {
	if len(raw) == 0 || len(raw) > domain.RichMaxPayloadBytes {
		return domain.RichPayload{}, ErrInvalidInput
	}
	var probe struct {
		Embeds      json.RawMessage `json:"embeds"`
		Attachments json.RawMessage `json:"attachments"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return domain.RichPayload{}, ErrInvalidInput
	}
	if len(probe.Embeds) > 0 && string(probe.Embeds) != "null" {
		var native domain.RichPayload
		if err := json.Unmarshal(raw, &native); err != nil {
			return domain.RichPayload{}, ErrInvalidInput
		}
		return ValidateRichPayload(native)
	}
	return MapSlackWebhook(raw)
}

func ValidateRichPayload(in domain.RichPayload) (domain.RichPayload, error) {
	out := domain.RichPayload{
		Text:     strings.TrimSpace(in.Text),
		Username: strings.TrimSpace(in.Username),
		IconURL:  sanitiseHTTPURL(in.IconURL),
	}
	if utf8.RuneCountInString(out.Text) > domain.RichMaxTextLength {
		return domain.RichPayload{}, ErrInvalidInput
	}
	if utf8.RuneCountInString(out.Username) > domain.RichMaxUsernameLength {
		return domain.RichPayload{}, ErrInvalidInput
	}
	if len(in.Embeds) > domain.RichMaxEmbeds {
		return domain.RichPayload{}, ErrInvalidInput
	}
	for _, embed := range in.Embeds {
		clean, ok := validateRichEmbed(embed)
		if !ok {
			return domain.RichPayload{}, ErrInvalidInput
		}
		if richEmbedEmpty(clean) {
			continue
		}
		out.Embeds = append(out.Embeds, clean)
	}
	if out.Text == "" && len(out.Embeds) == 0 {
		return domain.RichPayload{}, ErrInvalidInput
	}
	return out, nil
}

func FallbackBody(payload domain.RichPayload) string {
	if text := strings.TrimSpace(payload.Text); text != "" {
		if utf8.RuneCountInString(text) > 8000 {
			return string([]rune(text)[:8000])
		}
		return text
	}
	if len(payload.Embeds) == 0 {
		return "Incoming webhook"
	}
	embed := payload.Embeds[0]
	if embed.Title != "" {
		return embed.Title
	}
	if embed.Description != "" {
		if utf8.RuneCountInString(embed.Description) > 8000 {
			return string([]rune(embed.Description)[:8000])
		}
		return embed.Description
	}
	return "Incoming webhook"
}

func validateRichEmbed(in domain.RichEmbed) (domain.RichEmbed, bool) {
	out := domain.RichEmbed{
		Title:        clipRunes(strings.TrimSpace(in.Title), domain.RichMaxTitleLength),
		URL:          sanitiseHTTPURL(in.URL),
		Description:  clipRunes(strings.TrimSpace(in.Description), domain.RichMaxDescriptionLen),
		Color:        sanitiseColor(in.Color),
		ThumbnailURL: sanitiseHTTPURL(in.ThumbnailURL),
		ImageURL:     sanitiseHTTPURL(in.ImageURL),
	}
	if in.Timestamp != nil && !in.Timestamp.IsZero() {
		ts := in.Timestamp.UTC()
		out.Timestamp = &ts
	}
	if in.Author != nil {
		name := clipRunes(strings.TrimSpace(in.Author.Name), domain.RichMaxAuthorNameLen)
		if name != "" {
			out.Author = &domain.RichEmbedActor{
				Name:    name,
				URL:     sanitiseHTTPURL(in.Author.URL),
				IconURL: sanitiseHTTPURL(in.Author.IconURL),
			}
		}
	}
	if in.Footer != nil {
		text := clipRunes(strings.TrimSpace(in.Footer.Text), domain.RichMaxFooterLength)
		if text != "" {
			out.Footer = &domain.RichEmbedFooter{
				Text:    text,
				IconURL: sanitiseHTTPURL(in.Footer.IconURL),
			}
		}
	}
	if len(in.Fields) > domain.RichMaxFieldsPerEmbed {
		return domain.RichEmbed{}, false
	}
	for _, field := range in.Fields {
		name := clipRunes(strings.TrimSpace(field.Name), domain.RichMaxFieldNameLen)
		value := clipRunes(strings.TrimSpace(field.Value), domain.RichMaxFieldValueLen)
		if name == "" || value == "" {
			continue
		}
		out.Fields = append(out.Fields, domain.RichField{
			Name: name, Value: value, Inline: field.Inline,
		})
	}
	return out, true
}

func richEmbedEmpty(embed domain.RichEmbed) bool {
	return embed.Title == "" && embed.Description == "" && embed.URL == "" &&
		embed.ImageURL == "" && embed.ThumbnailURL == "" &&
		embed.Author == nil && embed.Footer == nil && len(embed.Fields) == 0
}

func sanitiseHTTPURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	if parsed.Host == "" {
		return ""
	}
	return parsed.String()
}

func sanitiseColor(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	switch strings.ToLower(raw) {
	case "good":
		return "#2eb886"
	case "warning":
		return "#daa038"
	case "danger":
		return "#a30200"
	}
	if strings.HasPrefix(raw, "#") {
		hex := raw[1:]
		if len(hex) != 3 && len(hex) != 6 {
			return ""
		}
		for _, c := range hex {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return ""
			}
		}
		return strings.ToLower(raw)
	}
	// Slack sends decimal colours such as "13632027".
	if len(raw) > 8 {
		return ""
	}
	n := 0
	for _, c := range raw {
		if c < '0' || c > '9' {
			return ""
		}
		n = n*10 + int(c-'0')
	}
	if n < 0 || n > 0xffffff {
		return ""
	}
	return "#" + strings.ToLower(padHex(n, 6))
}

func padHex(n int, width int) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, width)
	for i := width - 1; i >= 0; i-- {
		out[i] = hexdigits[n&0xf]
		n >>= 4
	}
	return string(out)
}

func clipRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}

func parseSlackTimestamp(raw any) *time.Time {
	switch v := raw.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			t = t.UTC()
			return &t
		}
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			t = t.UTC()
			return &t
		}
	case float64:
		if v <= 0 {
			return nil
		}
		t := time.Unix(int64(v), 0).UTC()
		return &t
	case json.Number:
		n, err := v.Int64()
		if err != nil || n <= 0 {
			return nil
		}
		t := time.Unix(n, 0).UTC()
		return &t
	}
	return nil
}
