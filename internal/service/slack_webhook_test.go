package service

import (
	"strings"
	"testing"

	"github.com/kilobyteno/dagr-chat/internal/domain"
)

func TestMapSlackWebhookAttachment(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"text": "Deploy finished",
		"username": "GitHub",
		"icon_url": "https://example.com/github.png",
		"attachments": [{
			"color": "13632027",
			"title": "Production deploy",
			"title_link": "https://example.com/run/1",
			"pretext": "All jobs passed.",
			"text": "Commit abc123",
			"author_name": "alice",
			"author_link": "https://example.com/alice",
			"footer": "GitHub Actions",
			"ts": 1710000000,
			"fields": [{"title": "Commit", "value": "` + "`abc123`" + `", "short": true}]
		}]
	}`)
	payload, err := MapSlackWebhook(raw)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Text != "Deploy finished" || payload.Username != "GitHub" {
		t.Fatalf("payload = %+v", payload)
	}
	if !strings.HasPrefix(payload.IconURL, "https://example.com/") {
		t.Fatalf("icon = %q", payload.IconURL)
	}
	if len(payload.Embeds) != 1 {
		t.Fatalf("embeds = %d", len(payload.Embeds))
	}
	embed := payload.Embeds[0]
	if embed.Title != "Production deploy" || embed.URL != "https://example.com/run/1" {
		t.Fatalf("embed = %+v", embed)
	}
	if embed.Color != "#d0021b" {
		t.Fatalf("color = %q", embed.Color)
	}
	if !strings.Contains(embed.Description, "All jobs passed.") {
		t.Fatalf("description = %q", embed.Description)
	}
	if embed.Author == nil || embed.Author.Name != "alice" {
		t.Fatalf("author = %+v", embed.Author)
	}
	if len(embed.Fields) != 1 || !embed.Fields[0].Inline {
		t.Fatalf("fields = %+v", embed.Fields)
	}
	if embed.Timestamp == nil || embed.Timestamp.Unix() != 1710000000 {
		t.Fatalf("ts = %v", embed.Timestamp)
	}
}

func TestParseIncomingWebhookPrefersNativeEmbeds(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"text": "Hello",
		"embeds": [{"title": "Native card"}],
		"attachments": [{"title": "Slack card"}]
	}`)
	payload, err := ParseIncomingWebhookPayload(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload.Embeds) != 1 || payload.Embeds[0].Title != "Native card" {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestParseIncomingWebhookTextOnly(t *testing.T) {
	t.Parallel()
	payload, err := ParseIncomingWebhookPayload([]byte(`{"text":"Just text"}`))
	if err != nil {
		t.Fatal(err)
	}
	if payload.Text != "Just text" || len(payload.Embeds) != 0 {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestValidateRichPayloadRejectsEmpty(t *testing.T) {
	t.Parallel()
	if _, err := ParseIncomingWebhookPayload([]byte(`{}`)); err == nil {
		t.Fatal("expected invalid empty payload")
	}
}

func TestMapSlackNamedColors(t *testing.T) {
	t.Parallel()
	payload, err := MapSlackWebhook([]byte(`{
		"attachments": [{"title": "Alert", "color": "danger", "text": "down"}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(payload.Embeds) != 1 || payload.Embeds[0].Color != "#a30200" {
		t.Fatalf("color = %+v", payload.Embeds)
	}
}

func TestValidateRichPayloadHexColor(t *testing.T) {
	t.Parallel()
	payload, err := ValidateRichPayload(domain.RichPayload{
		Text: "x",
		Embeds: []domain.RichEmbed{{Title: "t", Color: "#22C55E"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if payload.Embeds[0].Color != "#22c55e" {
		t.Fatalf("color = %q", payload.Embeds[0].Color)
	}
}
