package http

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/id"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
)

type httpAppState struct {
	apps            map[string]postgres.AppRow
	installs        map[uuid.UUID]map[uuid.UUID]postgres.WorkspaceAppInstallRow
	channelInstalls map[uuid.UUID]postgres.ChannelAppInstallRow
	hooks           map[uuid.UUID]postgres.IncomingWebhookRow
	hooksByHash     map[string]uuid.UUID
	memberKinds     map[uuid.UUID]map[uuid.UUID]string
}

func newHTTPAppState() *httpAppState {
	appID := id.New()
	now := time.Now().UTC()
	return &httpAppState{
		apps: map[string]postgres.AppRow{
			domain.AppSlugIncomingWebhooks: {
				ID: appID, Slug: domain.AppSlugIncomingWebhooks,
				Name: "Incoming Webhooks", Description: "Post messages into a channel.",
				Origin: domain.AppOriginFirstParty,
				Capabilities: []string{domain.CapabilityIncomingWebhook},
				CreatedAt: now, UpdatedAt: now,
			},
		},
		installs:        map[uuid.UUID]map[uuid.UUID]postgres.WorkspaceAppInstallRow{},
		channelInstalls: map[uuid.UUID]postgres.ChannelAppInstallRow{},
		hooks:           map[uuid.UUID]postgres.IncomingWebhookRow{},
		hooksByHash:     map[string]uuid.UUID{},
		memberKinds:     map[uuid.UUID]map[uuid.UUID]string{},
	}
}

func (m *httpWorkspaceStore) appsState() *httpAppState {
	if m.appState == nil {
		m.appState = newHTTPAppState()
	}
	return m.appState
}

func (m *httpWorkspaceStore) ListApps(_ context.Context) ([]postgres.AppRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.appsState()
	out := make([]postgres.AppRow, 0, len(state.apps))
	for _, app := range state.apps {
		out = append(out, app)
	}
	return out, nil
}

func (m *httpWorkspaceStore) GetAppBySlug(_ context.Context, slug string) (postgres.AppRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	app, ok := m.appsState().apps[slug]
	if !ok {
		return postgres.AppRow{}, postgres.ErrNotFound
	}
	return app, nil
}

func (m *httpWorkspaceStore) CountWorkspaceAppInstalls(_ context.Context, workspaceID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.appsState().installs[workspaceID]), nil
}

func (m *httpWorkspaceStore) GetWorkspaceAppInstall(
	_ context.Context, workspaceID, appID uuid.UUID,
) (postgres.WorkspaceAppInstallRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.appsState().installs[workspaceID][appID]
	if !ok {
		return postgres.WorkspaceAppInstallRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *httpWorkspaceStore) ListWorkspaceAppInstalls(_ context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceAppInstallRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.WorkspaceAppInstallRow
	for _, row := range m.appsState().installs[workspaceID] {
		out = append(out, row)
	}
	return out, nil
}

func (m *httpWorkspaceStore) CreateAppUser(_ context.Context, displayName string) (postgres.UserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.users == nil {
		m.users = newMemStore()
	}
	now := time.Now().UTC()
	row := postgres.UserRow{
		ID: id.New(), DisplayName: displayName, EmailVerified: true,
		NotificationLevel: string(domain.NotifyMentions),
		Locale:            string(domain.DefaultLocale()),
		CreatedAt:         now, UpdatedAt: now,
	}
	m.users.byID[row.ID] = row
	return row, nil
}

func (m *httpWorkspaceStore) AddAppWorkspaceMember(_ context.Context, workspaceID, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members[workspaceID] == nil {
		m.members[workspaceID] = map[uuid.UUID]string{}
	}
	m.members[workspaceID][userID] = "member"
	displayName := "Incoming Webhook"
	if m.users != nil {
		if u, ok := m.users.byID[userID]; ok && u.DisplayName != "" {
			displayName = u.DisplayName
		}
	}
	m.allocateHandleLocked(workspaceID, userID, displayName)
	state := m.appsState()
	if state.memberKinds[workspaceID] == nil {
		state.memberKinds[workspaceID] = map[uuid.UUID]string{}
	}
	state.memberKinds[workspaceID][userID] = domain.MemberKindApp
	return nil
}

func (m *httpWorkspaceStore) InsertWorkspaceAppInstall(
	_ context.Context, workspaceID, appID, installedBy, botUserID uuid.UUID,
) (postgres.WorkspaceAppInstallRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.appsState()
	if state.installs[workspaceID] == nil {
		state.installs[workspaceID] = map[uuid.UUID]postgres.WorkspaceAppInstallRow{}
	}
	if _, ok := state.installs[workspaceID][appID]; ok {
		return postgres.WorkspaceAppInstallRow{}, postgres.ErrConflict
	}
	var app postgres.AppRow
	for _, item := range state.apps {
		if item.ID == appID {
			app = item
			break
		}
	}
	by := installedBy
	row := postgres.WorkspaceAppInstallRow{
		ID: id.New(), WorkspaceID: workspaceID, AppID: appID,
		AppSlug: app.Slug, AppName: app.Name, InstalledBy: &by,
		BotUserID: botUserID, CreatedAt: time.Now().UTC(),
	}
	state.installs[workspaceID][appID] = row
	return row, nil
}

func (m *httpWorkspaceStore) DeleteWorkspaceAppInstall(_ context.Context, workspaceID, appID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.appsState()
	install, ok := state.installs[workspaceID][appID]
	if !ok {
		return postgres.ErrNotFound
	}
	for id, channelInstall := range state.channelInstalls {
		if channelInstall.WorkspaceAppInstallID == install.ID {
			delete(state.channelInstalls, id)
		}
	}
	for id, hook := range state.hooks {
		if hook.WorkspaceAppInstallID == install.ID {
			delete(state.hooksByHash, hook.TokenHash)
			delete(state.hooks, id)
		}
	}
	delete(state.installs[workspaceID], appID)
	return nil
}

func (m *httpWorkspaceStore) InsertChannelAppInstall(
	_ context.Context, workspaceInstallID, channelID uuid.UUID,
) (postgres.ChannelAppInstallRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch, ok := m.channelsByID[channelID]
	if !ok {
		return postgres.ChannelAppInstallRow{}, postgres.ErrNotFound
	}
	state := m.appsState()
	for _, existing := range state.channelInstalls {
		if existing.WorkspaceAppInstallID == workspaceInstallID && existing.ChannelID == channelID {
			return postgres.ChannelAppInstallRow{}, postgres.ErrConflict
		}
	}
	row := postgres.ChannelAppInstallRow{
		ID: id.New(), WorkspaceAppInstallID: workspaceInstallID, ChannelID: channelID,
		ChannelName: ch.Name, ChannelIsPrivate: ch.IsPrivate, ChannelIsDM: ch.Kind == "dm",
		CreatedAt: time.Now().UTC(),
	}
	state.channelInstalls[row.ID] = row
	return row, nil
}

func (m *httpWorkspaceStore) GetChannelAppInstallForApp(
	_ context.Context, workspaceID, channelID, appID uuid.UUID,
) (postgres.ChannelAppInstallRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.appsState()
	install, ok := state.installs[workspaceID][appID]
	if !ok {
		return postgres.ChannelAppInstallRow{}, postgres.ErrNotFound
	}
	for _, row := range state.channelInstalls {
		if row.WorkspaceAppInstallID == install.ID && row.ChannelID == channelID {
			return row, nil
		}
	}
	return postgres.ChannelAppInstallRow{}, postgres.ErrNotFound
}

func (m *httpWorkspaceStore) DeleteChannelAppInstall(_ context.Context, installID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.appsState()
	if _, ok := state.channelInstalls[installID]; !ok {
		return postgres.ErrNotFound
	}
	for id, hook := range state.hooks {
		if hook.ChannelAppInstallID == installID {
			delete(state.hooksByHash, hook.TokenHash)
			delete(state.hooks, id)
		}
	}
	delete(state.channelInstalls, installID)
	return nil
}

func (m *httpWorkspaceStore) InsertIncomingWebhook(
	_ context.Context, channelInstallID uuid.UUID, tokenHash, tokenPrefix string,
) (postgres.IncomingWebhookRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.appsState()
	channelInstall, ok := state.channelInstalls[channelInstallID]
	if !ok {
		return postgres.IncomingWebhookRow{}, postgres.ErrNotFound
	}
	ch := m.channelsByID[channelInstall.ChannelID]
	row := postgres.IncomingWebhookRow{
		ID: id.New(), ChannelAppInstallID: channelInstallID,
		WorkspaceAppInstallID: channelInstall.WorkspaceAppInstallID,
		ChannelID: channelInstall.ChannelID, ChannelName: ch.Name,
		ChannelIsPrivate: ch.IsPrivate, TokenHash: tokenHash, TokenPrefix: tokenPrefix,
		CreatedAt: time.Now().UTC(),
	}
	state.hooks[row.ID] = row
	state.hooksByHash[tokenHash] = row.ID
	return row, nil
}

func (m *httpWorkspaceStore) GetIncomingWebhookByChannelInstall(
	_ context.Context, channelInstallID uuid.UUID,
) (postgres.IncomingWebhookRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, hook := range m.appsState().hooks {
		if hook.ChannelAppInstallID == channelInstallID {
			return hook, nil
		}
	}
	return postgres.IncomingWebhookRow{}, postgres.ErrNotFound
}

func (m *httpWorkspaceStore) ListIncomingWebhooksForWorkspace(
	_ context.Context, workspaceID uuid.UUID,
) ([]postgres.IncomingWebhookRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.appsState()
	var out []postgres.IncomingWebhookRow
	for _, hook := range state.hooks {
		for _, install := range state.installs[workspaceID] {
			if hook.WorkspaceAppInstallID == install.ID {
				out = append(out, hook)
			}
		}
	}
	return out, nil
}

func (m *httpWorkspaceStore) GetIncomingWebhookByTokenHash(_ context.Context, tokenHash string) (postgres.IncomingWebhookRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.appsState().hooksByHash[tokenHash]
	if !ok {
		return postgres.IncomingWebhookRow{}, postgres.ErrNotFound
	}
	return m.appsState().hooks[id], nil
}

func (m *httpWorkspaceStore) GetIncomingWebhookBot(_ context.Context, webhookID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	hook, ok := m.appsState().hooks[webhookID]
	if !ok {
		return uuid.Nil, uuid.Nil, postgres.ErrNotFound
	}
	for workspaceID, installs := range m.appsState().installs {
		for _, install := range installs {
			if install.ID == hook.WorkspaceAppInstallID {
				return install.BotUserID, workspaceID, nil
			}
		}
	}
	return uuid.Nil, uuid.Nil, postgres.ErrNotFound
}

func (m *httpWorkspaceStore) RotateIncomingWebhook(
	_ context.Context, webhookID uuid.UUID, tokenHash, tokenPrefix string,
) (postgres.IncomingWebhookRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.appsState()
	hook, ok := state.hooks[webhookID]
	if !ok {
		return postgres.IncomingWebhookRow{}, postgres.ErrNotFound
	}
	delete(state.hooksByHash, hook.TokenHash)
	hook.TokenHash = tokenHash
	hook.TokenPrefix = tokenPrefix
	state.hooks[webhookID] = hook
	state.hooksByHash[tokenHash] = webhookID
	return hook, nil
}

func (m *httpWorkspaceStore) TouchIncomingWebhook(_ context.Context, webhookID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	hook, ok := m.appsState().hooks[webhookID]
	if !ok {
		return postgres.ErrNotFound
	}
	now := time.Now().UTC()
	hook.LastUsedAt = &now
	m.appsState().hooks[webhookID] = hook
	return nil
}
