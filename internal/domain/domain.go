// Package domain defines core domain types for Dagr.
package domain

import (
	"strings"
	"time"
)

// User is a registered account.
type User struct {
	ID                string
	Email             string
	DisplayName       string
	NotificationLevel NotificationLevel
	Locale            Locale
	EmailVerified     bool
	EmailVerifiedAt   *time.Time
	StatusEmoji       string
	StatusText        string
	StatusExpiresAt   *time.Time
	HasAvatar         bool
	AvatarUpdatedAt   *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// EffectiveCustomStatus returns emoji/text/expiry, clearing them when expired.
func EffectiveCustomStatus(emoji, text string, expiresAt *time.Time) (string, string, *time.Time) {
	if expiresAt != nil && !expiresAt.After(time.Now().UTC()) {
		return "", "", nil
	}
	if strings.TrimSpace(emoji) == "" && strings.TrimSpace(text) == "" {
		return "", "", nil
	}
	return emoji, text, expiresAt
}

// PresenceState is a user's live availability.
type PresenceState string

const (
	PresenceOnline  PresenceState = "online"
	PresenceAway    PresenceState = "away"
	PresenceOffline PresenceState = "offline"
)

// WorkspaceRole is a member's role within a workspace.
type WorkspaceRole string

const (
	WorkspaceRoleOwner  WorkspaceRole = "owner"
	WorkspaceRoleAdmin  WorkspaceRole = "admin"
	WorkspaceRoleMember WorkspaceRole = "member"
)

// Workspace is a tenant container for channels and members on a server.
type Workspace struct {
	ID            string
	Name          string
	Slug          string
	Role          WorkspaceRole // caller's role when listed for a user
	HasIcon       bool
	IconUpdatedAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// WorkspaceMember is a user's membership profile within a workspace.
type WorkspaceMember struct {
	UserID                string
	DisplayName           string
	Handle                string
	FormerHandles         []string
	Role                  WorkspaceRole
	Kind                  string // member | external
	IsExternal            bool
	HomeWorkspaceID       string
	HomeWorkspaceName     string
	HomeServerID          string
	HomeWorkspaceRemoteID string
	HomeWorkspaceIconURL  string
	HomeServerURL         string
	StatusEmoji           string
	StatusText            string
	StatusExpiresAt       *time.Time
	Presence              PresenceState
	HasAvatar             bool
	AvatarUpdatedAt       *time.Time
}

// ChannelMemberRole is a member's role within a private channel.
type ChannelMemberRole string

const (
	ChannelMemberRoleAdmin  ChannelMemberRole = "admin"
	ChannelMemberRoleMember ChannelMemberRole = "member"
)

// Channel is a conversation space within a workspace.
type Channel struct {
	ID                   string
	WorkspaceID          string
	Name                 string
	Topic                string
	IsPrivate            bool
	IsDM                 bool
	IsShared             bool
	CreatedBy            string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	UnreadCount          int
	FirstUnreadMessageID string
	// Peer fields are populated for 1:1 DMs (the other participant).
	PeerUserID          string
	PeerDisplayName     string
	PeerHandle          string
	PeerHasAvatar       bool
	PeerAvatarUpdatedAt *time.Time
}

// WorkspaceInvite is a pending (or accepted) invitation to join a workspace.
type WorkspaceInvite struct {
	ID          string
	WorkspaceID string
	Email       string
	Token       string
	Role        WorkspaceRole
	InvitedBy   string
	ExpiresAt   time.Time
	AcceptedAt  *time.Time
	CreatedAt   time.Time
}

// ScheduledMessageStatus is the lifecycle state of a scheduled message.
type ScheduledMessageStatus string

const (
	ScheduledPending   ScheduledMessageStatus = "pending"
	ScheduledSent      ScheduledMessageStatus = "sent"
	ScheduledCancelled ScheduledMessageStatus = "cancelled"
	ScheduledFailed    ScheduledMessageStatus = "failed"
)

// ScheduledMessage is a message waiting to be published at send_at.
type ScheduledMessage struct {
	ID            string
	ChannelID     string
	AuthorID      string
	Body          string
	ContentType   string
	SendAt        time.Time
	Status        ScheduledMessageStatus
	SentMessageID *string
	Error         string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// WorkspaceDomain is a DNS-verified (or pending) email domain claim for a workspace.
type WorkspaceDomain struct {
	ID                string
	WorkspaceID       string
	Domain            string
	VerificationToken string
	VerifiedAt        *time.Time
	AutoJoin          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Verified returns true when DNS verification has succeeded.
func (d WorkspaceDomain) Verified() bool {
	return d.VerifiedAt != nil
}

// NotificationKind is the type of in-app notification.
type NotificationKind string

const (
	NotificationMention         NotificationKind = "mention"
	NotificationMessage         NotificationKind = "message"
	NotificationReaction        NotificationKind = "reaction"
	NotificationChannelInvite   NotificationKind = "channel_invite"
	NotificationWorkspaceInvite NotificationKind = "workspace_invite"
	NotificationWorkspaceJoin   NotificationKind = "workspace_join"
)

// NotificationLevel controls how aggressively a user is notified.
// Account-level preference is the ceiling; channel preference can only reduce it.
type NotificationLevel string

const (
	NotifyAll      NotificationLevel = "all"
	NotifyMentions NotificationLevel = "mentions"
	NotifyNothing  NotificationLevel = "nothing"
)

// ChannelNotificationLevel is a per-channel override under the account level.
type ChannelNotificationLevel = NotificationLevel

const (
	ChannelNotifyAll      = NotifyAll
	ChannelNotifyMentions = NotifyMentions
	ChannelNotifyNothing  = NotifyNothing
)

// Locale is a supported UI language for the desktop client.
type Locale string

const (
	LocaleEnGB Locale = "en-GB"
	LocaleNb   Locale = "nb"
)

// DefaultLocale is British English.
func DefaultLocale() Locale {
	return LocaleEnGB
}

// ParseLocale validates a supported UI locale.
func ParseLocale(value string) (Locale, bool) {
	switch Locale(value) {
	case LocaleEnGB, LocaleNb:
		return Locale(value), true
	default:
		return "", false
	}
}

// ParseNotificationLevel validates a preference string.
func ParseNotificationLevel(value string) (NotificationLevel, bool) {
	switch NotificationLevel(value) {
	case NotifyAll, NotifyMentions, NotifyNothing:
		return NotificationLevel(value), true
	default:
		return "", false
	}
}

// ParseChannelNotificationLevel validates a preference string.
func ParseChannelNotificationLevel(value string) (ChannelNotificationLevel, bool) {
	return ParseNotificationLevel(value)
}

// MinNotificationLevel returns the more restrictive of two levels.
func MinNotificationLevel(a, b NotificationLevel) NotificationLevel {
	if notificationLevelRank(a) <= notificationLevelRank(b) {
		return a
	}
	return b
}

func notificationLevelRank(level NotificationLevel) int {
	switch level {
	case NotifyNothing:
		return 0
	case NotifyMentions:
		return 1
	case NotifyAll:
		return 2
	default:
		return 1
	}
}

// Notification is an inbox item for a user.
type Notification struct {
	ID          string
	UserID      string
	ActorID     string
	ActorName   string
	Kind        NotificationKind
	WorkspaceID string
	ChannelID   string
	ChannelName string
	IsDM        bool
	MessageID   string
	Body        string
	ReadAt      *time.Time
	CreatedAt   time.Time
}

// Message content types.
const (
	ContentTypePlain  = "text/plain"
	ContentTypeSystem = "application/x-dagr-system"
)

// LinkPreviewStatus is the lifecycle of a message URL unfurl.
type LinkPreviewStatus string

const (
	LinkPreviewPending LinkPreviewStatus = "pending"
	LinkPreviewReady   LinkPreviewStatus = "ready"
	LinkPreviewFailed  LinkPreviewStatus = "failed"
	LinkPreviewSkipped LinkPreviewStatus = "skipped"
)

// LinkPreview is rich metadata for a URL found in a message body.
type LinkPreview struct {
	ID            string
	MessageID     string
	URL           string
	NormalizedURL string
	Status        LinkPreviewStatus
	Title         string
	Description   string
	SiteName      string
	ImageURL      string
	FetchedAt     *time.Time
	CreatedAt     time.Time
}

// MessageReaction is an aggregated emoji reaction on a message.
type MessageReaction struct {
	Emoji   string
	Count   int
	Reacted bool
	UserIDs []string
}

// Message is a chat message. Ciphertext may be set when E2EE is enabled for DMs.
type Message struct {
	ID                  string
	ChannelID           string
	AuthorID            string
	AuthorName          string // optional display name when listed
	AuthorHandle        string // workspace-scoped handle when listed
	AuthorHasAvatar     bool
	AuthorAvatarUpdated *time.Time
	Body                string // plaintext when E2EE is off
	Ciphertext          []byte // optional encrypted payload for E2EE DMs
	ContentType         string // e.g. text/plain, application/x-dagr-system
	LinkPreviews        []LinkPreview
	Reactions           []MessageReaction
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// FileAttachment metadata for an uploaded object in object storage.
type FileAttachment struct {
	ID          string
	UploaderID  string
	ObjectKey   string
	Filename    string
	ContentType string
	SizeBytes   int64
	CreatedAt   time.Time
}
