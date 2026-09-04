package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/config"
	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/id"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

func (m *chatMemStore) ListApps(_ context.Context) ([]postgres.AppRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]postgres.AppRow, 0, len(m.appCatalog))
	for _, app := range m.appCatalog {
		out = append(out, app)
	}
	return out, nil
}

func (m *chatMemStore) GetAppBySlug(_ context.Context, slug string) (postgres.AppRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	app, ok := m.appCatalog[slug]
	if !ok {
		return postgres.AppRow{}, postgres.ErrNotFound
	}
	return app, nil
}

func (m *chatMemStore) CountWorkspaceAppInstalls(_ context.Context, workspaceID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.appInstalls[workspaceID]), nil
}

func (m *chatMemStore) GetWorkspaceAppInstall(_ context.Context, workspaceID, appID uuid.UUID) (postgres.WorkspaceAppInstallRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.appInstalls[workspaceID][appID]
	if !ok {
		return postgres.WorkspaceAppInstallRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *chatMemStore) ListWorkspaceAppInstalls(_ context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceAppInstallRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.WorkspaceAppInstallRow
	for _, row := range m.appInstalls[workspaceID] {
		out = append(out, row)
	}
	return out, nil
}

func (m *chatMemStore) CreateAppUser(_ context.Context, displayName string) (postgres.UserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	row := postgres.UserRow{
		ID: id.New(), DisplayName: displayName, EmailVerified: true,
		NotificationLevel: string(domain.NotifyMentions),
		CreatedAt: now, UpdatedAt: now,
	}
	m.usersByID[row.ID] = row
	return row, nil
}

func (m *chatMemStore) AddAppWorkspaceMember(_ context.Context, workspaceID, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members[workspaceID] == nil {
		m.members[workspaceID] = map[uuid.UUID]string{}
	}
	m.members[workspaceID][userID] = "member"
	displayName := domain.IncomingWebhookBotName
	if u, ok := m.usersByID[userID]; ok && u.DisplayName != "" {
		displayName = u.DisplayName
	}
	m.allocateHandleLocked(workspaceID, userID, displayName)
	if m.memberKinds[workspaceID] == nil {
		m.memberKinds[workspaceID] = map[uuid.UUID]string{}
	}
	m.memberKinds[workspaceID][userID] = domain.MemberKindApp
	return nil
}

func (m *chatMemStore) InsertWorkspaceAppInstall(
	_ context.Context, workspaceID, appID, installedBy, botUserID uuid.UUID,
) (postgres.WorkspaceAppInstallRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.appInstalls[workspaceID] == nil {
		m.appInstalls[workspaceID] = map[uuid.UUID]postgres.WorkspaceAppInstallRow{}
	}
	if _, ok := m.appInstalls[workspaceID][appID]; ok {
		return postgres.WorkspaceAppInstallRow{}, postgres.ErrConflict
	}
	var app postgres.AppRow
	for _, item := range m.appCatalog {
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
	m.appInstalls[workspaceID][appID] = row
	return row, nil
}

func (m *chatMemStore) DeleteWorkspaceAppInstall(_ context.Context, workspaceID, appID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	install, ok := m.appInstalls[workspaceID][appID]
	if !ok {
		return postgres.ErrNotFound
	}
	for id, channelInstall := range m.channelAppInstalls {
		if channelInstall.WorkspaceAppInstallID == install.ID {
			delete(m.channelAppInstalls, id)
		}
	}
	for id, hook := range m.incomingHooks {
		if hook.WorkspaceAppInstallID == install.ID {
			delete(m.hooksByHash, hook.TokenHash)
			delete(m.incomingHooks, id)
		}
	}
	delete(m.appInstalls[workspaceID], appID)
	return nil
}

func (m *chatMemStore) InsertChannelAppInstall(
	_ context.Context, workspaceInstallID, channelID uuid.UUID,
) (postgres.ChannelAppInstallRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch, ok := m.channelsByID[channelID]
	if !ok {
		return postgres.ChannelAppInstallRow{}, postgres.ErrNotFound
	}
	for _, existing := range m.channelAppInstalls {
		if existing.WorkspaceAppInstallID == workspaceInstallID && existing.ChannelID == channelID {
			return postgres.ChannelAppInstallRow{}, postgres.ErrConflict
		}
	}
	row := postgres.ChannelAppInstallRow{
		ID: id.New(), WorkspaceAppInstallID: workspaceInstallID, ChannelID: channelID,
		ChannelName: ch.Name, ChannelIsPrivate: ch.IsPrivate, ChannelIsDM: ch.Kind == "dm",
		CreatedAt: time.Now().UTC(),
	}
	m.channelAppInstalls[row.ID] = row
	return row, nil
}

func (m *chatMemStore) GetChannelAppInstallForApp(
	_ context.Context, workspaceID, channelID, appID uuid.UUID,
) (postgres.ChannelAppInstallRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	install, ok := m.appInstalls[workspaceID][appID]
	if !ok {
		return postgres.ChannelAppInstallRow{}, postgres.ErrNotFound
	}
	for _, row := range m.channelAppInstalls {
		if row.WorkspaceAppInstallID == install.ID && row.ChannelID == channelID {
			return row, nil
		}
	}
	return postgres.ChannelAppInstallRow{}, postgres.ErrNotFound
}

func (m *chatMemStore) DeleteChannelAppInstall(_ context.Context, installID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.channelAppInstalls[installID]; !ok {
		return postgres.ErrNotFound
	}
	for id, hook := range m.incomingHooks {
		if hook.ChannelAppInstallID == installID {
			delete(m.hooksByHash, hook.TokenHash)
			delete(m.incomingHooks, id)
		}
	}
	delete(m.channelAppInstalls, installID)
	return nil
}

func (m *chatMemStore) InsertIncomingWebhook(
	_ context.Context, channelInstallID uuid.UUID, tokenHash, tokenPrefix string,
) (postgres.IncomingWebhookRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	channelInstall, ok := m.channelAppInstalls[channelInstallID]
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
	m.incomingHooks[row.ID] = row
	m.hooksByHash[tokenHash] = row.ID
	return row, nil
}

func (m *chatMemStore) GetIncomingWebhookByChannelInstall(
	_ context.Context, channelInstallID uuid.UUID,
) (postgres.IncomingWebhookRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, hook := range m.incomingHooks {
		if hook.ChannelAppInstallID == channelInstallID {
			return hook, nil
		}
	}
	return postgres.IncomingWebhookRow{}, postgres.ErrNotFound
}

func (m *chatMemStore) ListIncomingWebhooksForWorkspace(_ context.Context, workspaceID uuid.UUID) ([]postgres.IncomingWebhookRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.IncomingWebhookRow
	for _, hook := range m.incomingHooks {
		for _, install := range m.appInstalls[workspaceID] {
			if hook.WorkspaceAppInstallID == install.ID {
				out = append(out, hook)
			}
		}
	}
	return out, nil
}

func (m *chatMemStore) GetIncomingWebhookByTokenHash(_ context.Context, tokenHash string) (postgres.IncomingWebhookRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.hooksByHash[tokenHash]
	if !ok {
		return postgres.IncomingWebhookRow{}, postgres.ErrNotFound
	}
	return m.incomingHooks[id], nil
}

func (m *chatMemStore) GetIncomingWebhookBot(_ context.Context, webhookID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	hook, ok := m.incomingHooks[webhookID]
	if !ok {
		return uuid.Nil, uuid.Nil, postgres.ErrNotFound
	}
	for workspaceID, installs := range m.appInstalls {
		for _, install := range installs {
			if install.ID == hook.WorkspaceAppInstallID {
				return install.BotUserID, workspaceID, nil
			}
		}
	}
	return uuid.Nil, uuid.Nil, postgres.ErrNotFound
}

func (m *chatMemStore) RotateIncomingWebhook(
	_ context.Context, webhookID uuid.UUID, tokenHash, tokenPrefix string,
) (postgres.IncomingWebhookRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	hook, ok := m.incomingHooks[webhookID]
	if !ok {
		return postgres.IncomingWebhookRow{}, postgres.ErrNotFound
	}
	delete(m.hooksByHash, hook.TokenHash)
	hook.TokenHash = tokenHash
	hook.TokenPrefix = tokenPrefix
	m.incomingHooks[webhookID] = hook
	m.hooksByHash[tokenHash] = webhookID
	return hook, nil
}

func (m *chatMemStore) TouchIncomingWebhook(_ context.Context, webhookID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	hook, ok := m.incomingHooks[webhookID]
	if !ok {
		return postgres.ErrNotFound
	}
	now := time.Now().UTC()
	hook.LastUsedAt = &now
	m.incomingHooks[webhookID] = hook
	return nil
}

func newAppServices() (*chatMemStore, *service.AppService, *service.WebhookService, *service.ChannelService) {
	store := newChatMemStore()
	channels := service.NewChannelService(store)
	messages := service.NewMessageService(store, channels)
	apps := service.NewAppService(store)
	cfg := config.Config{ServerPublicURL: "http://localhost:8383"}
	webhooks := service.NewWebhookService(store, apps, messages, cfg)
	return store, apps, webhooks, channels
}

func TestIncomingWebhookEnableRotateAndPost(t *testing.T) {
	store, apps, webhooks, channels := newAppServices()
	owner := store.seedUser("owner@example.com", "Owner")
	ws, ch := store.seedWorkspace(owner.ID, "acme")
	ctx := context.Background()

	if _, err := apps.Install(ctx, owner.ID.String(), ws.ID.String(), domain.AppSlugIncomingWebhooks); err != nil {
		t.Fatal(err)
	}
	secret, err := webhooks.EnableOnChannel(ctx, owner.ID.String(), ch.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if secret.Token == "" || !strings.Contains(secret.URL, "/api/v1/hooks/") {
		t.Fatalf("secret = %+v", secret)
	}

	msg, err := webhooks.Receive(ctx, secret.Token, []byte(`{"text":"Hello from CI"}`))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Body != "Hello from CI" || msg.ContentType != domain.ContentTypeRich {
		t.Fatalf("message = %+v", msg)
	}

	rotated, err := webhooks.RotateOnChannel(ctx, owner.ID.String(), ch.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := webhooks.Receive(ctx, secret.Token, []byte(`{"text":"old"}`)); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("old token: %v", err)
	}
	if _, err := webhooks.Receive(ctx, rotated.Token, []byte(`{"text":"new"}`)); err != nil {
		t.Fatal(err)
	}

	members, err := channels.ListMembers(ctx, owner.ID.String(), ch.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range members {
		if member.Kind == domain.MemberKindApp {
			t.Fatal("app users should not appear in People")
		}
	}
}

func TestIncomingWebhookAdminOnlyAndDMReject(t *testing.T) {
	store, _, webhooks, _ := newAppServices()
	owner := store.seedUser("owner@example.com", "Owner")
	member := store.seedUser("member@example.com", "Member")
	ws, ch := store.seedWorkspace(owner.ID, "acme")
	_ = store.AddWorkspaceMember(context.Background(), ws.ID, member.ID, domain.WorkspaceRoleMember)
	ctx := context.Background()

	if _, err := webhooks.EnableOnChannel(ctx, member.ID.String(), ch.ID.String()); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("member enable: %v", err)
	}

	peer := store.seedUser("peer@example.com", "Peer")
	_ = store.AddWorkspaceMember(ctx, ws.ID, peer.ID, domain.WorkspaceRoleMember)
	dm, err := store.CreateDMChannel(ctx, ws.ID, owner.ID, peer.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := webhooks.EnableOnChannel(ctx, owner.ID.String(), dm.ID.String()); !errors.Is(err, service.ErrNotAChannel) {
		t.Fatalf("dm enable: %v", err)
	}
}

func TestIncomingWebhookAppLimit(t *testing.T) {
	store := newChatMemStore()
	apps := service.NewAppService(store).WithEntitlements(fixedEntitlements{maxApps: 0})
	owner := store.seedUser("owner@example.com", "Owner")
	ws, _ := store.seedWorkspace(owner.ID, "acme")
	if _, err := apps.Install(context.Background(), owner.ID.String(), ws.ID.String(), domain.AppSlugIncomingWebhooks); !errors.Is(err, service.ErrAppLimit) {
		t.Fatalf("limit: %v", err)
	}
}

func TestIncomingWebhookRateLimit(t *testing.T) {
	store, apps, webhooks, _ := newAppServices()
	owner := store.seedUser("owner@example.com", "Owner")
	ws, ch := store.seedWorkspace(owner.ID, "acme")
	ctx := context.Background()
	if _, err := apps.Install(ctx, owner.ID.String(), ws.ID.String(), domain.AppSlugIncomingWebhooks); err != nil {
		t.Fatal(err)
	}
	secret, err := webhooks.EnableOnChannel(ctx, owner.ID.String(), ch.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < domain.IncomingWebhookRatePerMinute; i++ {
		if _, err := webhooks.Receive(ctx, secret.Token, []byte(`{"text":"n"}`)); err != nil {
			t.Fatalf("post %d: %v", i, err)
		}
	}
	if _, err := webhooks.Receive(ctx, secret.Token, []byte(`{"text":"over"}`)); !errors.Is(err, service.ErrWebhookRateLimited) {
		t.Fatalf("rate limit: %v", err)
	}
}

type fixedEntitlements struct {
	maxApps int
}

func (f fixedEntitlements) ForWorkspace(context.Context, string) domain.Entitlements {
	n := f.maxApps
	return domain.Entitlements{MaxApps: &n}
}
