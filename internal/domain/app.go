package domain

import "time"

// App origins distinguish first-party catalog entries from later custom apps.
const (
	AppOriginFirstParty = "first_party"
	AppOriginCustom     = "custom"
)

const (
	AppSlugIncomingWebhooks     = "incoming-webhooks"
	CapabilityIncomingWebhook   = "incoming_webhook"
	IncomingWebhookBotName      = "Incoming Webhook"
	IncomingWebhookRatePerMinute = 30
)

const (
	UserKindHuman = "human"
	UserKindApp   = "app"
	MemberKindApp = "app"
)

// App is a catalog entry that can be installed on a workspace.
type App struct {
	ID           string
	Slug         string
	Name         string
	Description  string
	Origin       string
	OwnerUserID  string
	Capabilities []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// WorkspaceAppInstall is an app enabled for a workspace.
type WorkspaceAppInstall struct {
	ID          string
	WorkspaceID string
	AppID       string
	AppSlug     string
	AppName     string
	InstalledBy string
	BotUserID   string
	CreatedAt   time.Time
}

// ChannelAppInstall is an app enabled on a specific channel.
type ChannelAppInstall struct {
	ID                   string
	WorkspaceAppInstallID string
	ChannelID            string
	ChannelName          string
	ChannelIsPrivate     bool
	CreatedAt            time.Time
}

// IncomingWebhook is the secret bound to a channel install.
type IncomingWebhook struct {
	ID               string
	ChannelID        string
	ChannelName      string
	ChannelIsPrivate bool
	TokenPrefix      string
	URL              string
	LastUsedAt       *time.Time
	CreatedAt        time.Time
}

// WorkspaceApp is a catalog app plus install state for one workspace.
type WorkspaceApp struct {
	App          App
	Installed    bool
	Install      *WorkspaceAppInstall
	ChannelHooks []IncomingWebhook
}
