package service

import (
	"encoding/json"
	"strings"

	"github.com/kilobyteno/dagr-chat/internal/domain"
)

type slackWebhookPayload struct {
	Text        string            `json:"text"`
	Username    string            `json:"username"`
	IconURL     string            `json:"icon_url"`
	Attachments []slackAttachment `json:"attachments"`
}

type slackAttachment struct {
	Color      string       `json:"color"`
	Pretext    string       `json:"pretext"`
	Text       string       `json:"text"`
	Title      string       `json:"title"`
	TitleLink  string       `json:"title_link"`
	Fallback   string       `json:"fallback"`
	AuthorName string       `json:"author_name"`
	AuthorLink string       `json:"author_link"`
	AuthorIcon string       `json:"author_icon"`
	Footer     string       `json:"footer"`
	FooterIcon string       `json:"footer_icon"`
	ImageURL   string       `json:"image_url"`
	ThumbURL   string       `json:"thumb_url"`
	TS         json.RawMessage `json:"ts"`
	Fields     []slackField `json:"fields"`
}

type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func MapSlackWebhook(raw []byte) (domain.RichPayload, error) {
	var in slackWebhookPayload
	if err := json.Unmarshal(raw, &in); err != nil {
		return domain.RichPayload{}, ErrInvalidInput
	}
	out := domain.RichPayload{
		Text:     strings.TrimSpace(in.Text),
		Username: strings.TrimSpace(in.Username),
		IconURL:  strings.TrimSpace(in.IconURL),
	}
	for _, att := range in.Attachments {
		embed := domain.RichEmbed{
			Title:        strings.TrimSpace(att.Title),
			URL:          strings.TrimSpace(att.TitleLink),
			Color:        strings.TrimSpace(att.Color),
			ImageURL:     strings.TrimSpace(att.ImageURL),
			ThumbnailURL: strings.TrimSpace(att.ThumbURL),
		}
		desc := strings.TrimSpace(att.Text)
		if pretext := strings.TrimSpace(att.Pretext); pretext != "" {
			if desc != "" {
				desc = pretext + "\n\n" + desc
			} else {
				desc = pretext
			}
		}
		if desc == "" {
			desc = strings.TrimSpace(att.Fallback)
		}
		embed.Description = desc
		if name := strings.TrimSpace(att.AuthorName); name != "" {
			embed.Author = &domain.RichEmbedActor{
				Name:    name,
				URL:     strings.TrimSpace(att.AuthorLink),
				IconURL: strings.TrimSpace(att.AuthorIcon),
			}
		}
		if footer := strings.TrimSpace(att.Footer); footer != "" {
			embed.Footer = &domain.RichEmbedFooter{
				Text:    footer,
				IconURL: strings.TrimSpace(att.FooterIcon),
			}
		}
		if len(att.TS) > 0 && string(att.TS) != "null" {
			var ts any
			if err := json.Unmarshal(att.TS, &ts); err == nil {
				embed.Timestamp = parseSlackTimestamp(ts)
			}
		}
		for _, field := range att.Fields {
			embed.Fields = append(embed.Fields, domain.RichField{
				Name:   strings.TrimSpace(field.Title),
				Value:  strings.TrimSpace(field.Value),
				Inline: field.Short,
			})
		}
		out.Embeds = append(out.Embeds, embed)
	}
	return ValidateRichPayload(out)
}
