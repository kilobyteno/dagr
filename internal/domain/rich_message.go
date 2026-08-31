package domain

import "time"

// Content type for structured incoming webhook posts.
const ContentTypeRich = "application/x-dagr-rich"

const (
	RichMaxEmbeds          = 10
	RichMaxFieldsPerEmbed  = 25
	RichMaxTitleLength     = 256
	RichMaxDescriptionLen  = 4096
	RichMaxFieldNameLen    = 256
	RichMaxFieldValueLen   = 1024
	RichMaxTextLength      = 8000
	RichMaxUsernameLength  = 80
	RichMaxPayloadBytes    = 32 << 10
	RichMaxFooterLength    = 256
	RichMaxAuthorNameLen   = 256
)

// RichPayload is the native Dagr incoming webhook body stored on a message.
type RichPayload struct {
	Text     string      `json:"text,omitempty"`
	Username string      `json:"username,omitempty"`
	IconURL  string      `json:"iconUrl,omitempty"`
	Embeds   []RichEmbed `json:"embeds,omitempty"`
}

// RichEmbed is a Discord-style card attached to a rich message.
type RichEmbed struct {
	Title        string          `json:"title,omitempty"`
	URL          string          `json:"url,omitempty"`
	Description  string          `json:"description,omitempty"`
	Color        string          `json:"color,omitempty"`
	Author       *RichEmbedActor `json:"author,omitempty"`
	Fields       []RichField     `json:"fields,omitempty"`
	ThumbnailURL string          `json:"thumbnailUrl,omitempty"`
	ImageURL     string          `json:"imageUrl,omitempty"`
	Footer       *RichEmbedFooter `json:"footer,omitempty"`
	Timestamp    *time.Time      `json:"timestamp,omitempty"`
}

// RichEmbedActor is the optional author line on an embed.
type RichEmbedActor struct {
	Name    string `json:"name,omitempty"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"iconUrl,omitempty"`
}

// RichField is a labelled value on an embed.
type RichField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// RichEmbedFooter is the small line under an embed.
type RichEmbedFooter struct {
	Text    string `json:"text,omitempty"`
	IconURL string `json:"iconUrl,omitempty"`
}
