package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
)

var (
	ErrAppLimit         = errors.New("app limit reached")
	ErrAlreadyInstalled = errors.New("already installed")
)

type AppStore interface {
	ListApps(ctx context.Context) ([]postgres.AppRow, error)
	GetAppBySlug(ctx context.Context, slug string) (postgres.AppRow, error)
	GetWorkspaceForUser(ctx context.Context, workspaceID, userID uuid.UUID) (postgres.WorkspaceRow, error)
	CountWorkspaceAppInstalls(ctx context.Context, workspaceID uuid.UUID) (int, error)
	GetWorkspaceAppInstall(ctx context.Context, workspaceID, appID uuid.UUID) (postgres.WorkspaceAppInstallRow, error)
	ListWorkspaceAppInstalls(ctx context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceAppInstallRow, error)
	ListIncomingWebhooksForWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]postgres.IncomingWebhookRow, error)
	CreateAppUser(ctx context.Context, displayName string) (postgres.UserRow, error)
	AddAppWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) error
	InsertWorkspaceAppInstall(ctx context.Context, workspaceID, appID, installedBy, botUserID uuid.UUID) (postgres.WorkspaceAppInstallRow, error)
	DeleteWorkspaceAppInstall(ctx context.Context, workspaceID, appID uuid.UUID) error
}

type AppService struct {
	store        AppStore
	entitlements EntitlementLookup
}

func NewAppService(store AppStore) *AppService {
	return &AppService{store: store}
}

func (s *AppService) WithEntitlements(lookup EntitlementLookup) *AppService {
	s.entitlements = lookup
	return s
}

func (s *AppService) ListForWorkspace(ctx context.Context, userID, workspaceID string) ([]domain.WorkspaceApp, error) {
	if _, _, err := s.requireManage(ctx, userID, workspaceID); err != nil {
		return nil, err
	}
	wid, _ := uuid.Parse(workspaceID)
	apps, err := s.store.ListApps(ctx)
	if err != nil {
		return nil, err
	}
	installs, err := s.store.ListWorkspaceAppInstalls(ctx, wid)
	if err != nil {
		return nil, err
	}
	hooks, err := s.store.ListIncomingWebhooksForWorkspace(ctx, wid)
	if err != nil {
		return nil, err
	}
	installByApp := map[uuid.UUID]postgres.WorkspaceAppInstallRow{}
	for _, install := range installs {
		installByApp[install.AppID] = install
	}
	hooksByInstall := map[uuid.UUID][]domain.IncomingWebhook{}
	for _, hook := range hooks {
		hooksByInstall[hook.WorkspaceAppInstallID] = append(hooksByInstall[hook.WorkspaceAppInstallID], hook.ToDomain())
	}

	out := make([]domain.WorkspaceApp, 0, len(apps))
	for _, app := range apps {
		item := domain.WorkspaceApp{App: app.ToDomain()}
		if install, ok := installByApp[app.ID]; ok {
			inst := install.ToDomain()
			item.Installed = true
			item.Install = &inst
			item.ChannelHooks = hooksByInstall[install.ID]
			if item.ChannelHooks == nil {
				item.ChannelHooks = []domain.IncomingWebhook{}
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *AppService) Install(ctx context.Context, userID, workspaceID, appSlug string) (*domain.WorkspaceAppInstall, error) {
	uid, wid, err := s.requireManage(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	app, err := s.store.GetAppBySlug(ctx, appSlug)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if existing, err := s.store.GetWorkspaceAppInstall(ctx, wid, app.ID); err == nil {
		inst := existing.ToDomain()
		return &inst, nil
	} else if !errors.Is(err, postgres.ErrNotFound) {
		return nil, err
	}
	if err := s.enforceAppLimit(ctx, wid); err != nil {
		return nil, err
	}
	bot, err := s.store.CreateAppUser(ctx, domain.IncomingWebhookBotName)
	if err != nil {
		return nil, err
	}
	if err := s.store.AddAppWorkspaceMember(ctx, wid, bot.ID); err != nil {
		return nil, err
	}
	row, err := s.store.InsertWorkspaceAppInstall(ctx, wid, app.ID, uid, bot.ID)
	if err != nil {
		if errors.Is(err, postgres.ErrConflict) {
			return nil, ErrAlreadyInstalled
		}
		return nil, err
	}
	inst := row.ToDomain()
	return &inst, nil
}

func (s *AppService) Uninstall(ctx context.Context, userID, workspaceID, appSlug string) error {
	_, wid, err := s.requireManage(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	app, err := s.store.GetAppBySlug(ctx, appSlug)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if err := s.store.DeleteWorkspaceAppInstall(ctx, wid, app.ID); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *AppService) EnsureInstalled(
	ctx context.Context, userID, workspaceID, appSlug string,
) (postgres.WorkspaceAppInstallRow, postgres.AppRow, error) {
	install, err := s.Install(ctx, userID, workspaceID, appSlug)
	if err != nil {
		return postgres.WorkspaceAppInstallRow{}, postgres.AppRow{}, err
	}
	app, err := s.store.GetAppBySlug(ctx, appSlug)
	if err != nil {
		return postgres.WorkspaceAppInstallRow{}, postgres.AppRow{}, err
	}
	wid, _ := uuid.Parse(workspaceID)
	row, err := s.store.GetWorkspaceAppInstall(ctx, wid, app.ID)
	if err != nil {
		return postgres.WorkspaceAppInstallRow{}, postgres.AppRow{}, err
	}
	_ = install
	return row, app, nil
}

func (s *AppService) requireManage(ctx context.Context, userID, workspaceID string) (uuid.UUID, uuid.UUID, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrNotFound
	}
	ws, err := s.store.GetWorkspaceForUser(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return uuid.Nil, uuid.Nil, ErrNotFound
		}
		return uuid.Nil, uuid.Nil, err
	}
	if !canManageWorkspace(domain.WorkspaceRole(ws.Role)) {
		return uuid.Nil, uuid.Nil, ErrForbidden
	}
	return uid, wid, nil
}

func (s *AppService) enforceAppLimit(ctx context.Context, workspaceID uuid.UUID) error {
	if s.entitlements == nil {
		return nil
	}
	ents := s.entitlements.ForWorkspace(ctx, workspaceID.String())
	if ents.MaxApps == nil {
		return nil
	}
	n, err := s.store.CountWorkspaceAppInstalls(ctx, workspaceID)
	if err != nil {
		return err
	}
	if n >= *ents.MaxApps {
		return ErrAppLimit
	}
	return nil
}
