package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
)

var (
	ErrForbidden     = errors.New("forbidden")
	ErrNotFound      = errors.New("not found")
	ErrWorkspaceName = errors.New("invalid workspace name")
	ErrInvalidIcon   = errors.New("invalid workspace icon")
)

const maxWorkspaceIconBytes = 2 << 20 // 2 MiB

var allowedWorkspaceIconTypes = map[string]string{
	"image/png":  "image/png",
	"image/jpeg": "image/jpeg",
	"image/jpg":  "image/jpeg",
	"image/webp": "image/webp",
	"image/gif":  "image/gif",
}

type WorkspaceStore interface {
	CreateWorkspace(ctx context.Context, name, slug string, createdBy uuid.UUID) (postgres.CreateWorkspaceResult, error)
	ListWorkspacesForUser(ctx context.Context, userID uuid.UUID) ([]postgres.WorkspaceRow, error)
	GetWorkspaceForUser(ctx context.Context, workspaceID, userID uuid.UUID) (postgres.WorkspaceRow, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
	SlugExistsExcluding(ctx context.Context, slug string, excludeID uuid.UUID) (bool, error)
	UpdateWorkspace(ctx context.Context, workspaceID uuid.UUID, name, slug string) (postgres.WorkspaceRow, error)
	SetWorkspaceIcon(ctx context.Context, workspaceID uuid.UUID, contentType string, data []byte) (postgres.WorkspaceRow, error)
	ClearWorkspaceIcon(ctx context.Context, workspaceID uuid.UUID) (postgres.WorkspaceRow, error)
	GetWorkspaceIcon(ctx context.Context, workspaceID uuid.UUID) (postgres.WorkspaceIcon, error)
	DeleteWorkspace(ctx context.Context, workspaceID uuid.UUID) error
	ListChannelsForWorkspace(ctx context.Context, workspaceID, userID uuid.UUID) ([]postgres.ChannelRow, error)
	GetWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) (postgres.WorkspaceMemberInfo, error)
	ListWorkspaceMembers(ctx context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceMemberInfo, error)
	MemberHandleExists(ctx context.Context, workspaceID uuid.UUID, handle string, excludeUserID uuid.UUID) (bool, error)
	UpdateWorkspaceMemberHandle(ctx context.Context, workspaceID, userID uuid.UUID, handle string) (postgres.WorkspaceMemberInfo, error)
	CountWorkspaceOwners(ctx context.Context, workspaceID uuid.UUID) (int, error)
	UpdateWorkspaceMemberRole(ctx context.Context, workspaceID, userID uuid.UUID, role domain.WorkspaceRole) (postgres.WorkspaceMemberInfo, error)
	RemoveWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) error
	TransferWorkspaceOwnership(ctx context.Context, workspaceID, fromOwner, toUser uuid.UUID) error
}

type WorkspaceLifecycle interface {
	OnWorkspaceCreated(ctx context.Context, workspaceID string) error
	OnWorkspaceDeleting(ctx context.Context, workspaceID string) error
	SyncSeats(ctx context.Context, workspaceID string) error
}

type WorkspaceService struct {
	store     WorkspaceStore
	lifecycle WorkspaceLifecycle
}

func NewWorkspaceService(store WorkspaceStore) *WorkspaceService {
	return &WorkspaceService{store: store}
}

func (s *WorkspaceService) WithLifecycle(lifecycle WorkspaceLifecycle) *WorkspaceService {
	s.lifecycle = lifecycle
	return s
}

type CreateWorkspaceResult struct {
	Workspace domain.Workspace
	Channels  []domain.Channel
}

func (s *WorkspaceService) Create(ctx context.Context, userID string, name string) (*CreateWorkspaceResult, error) {
	name = capitaliseName(strings.TrimSpace(name))
	if name == "" || len(name) > 80 {
		return nil, ErrWorkspaceName
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	baseSlug := slugify(name)
	if baseSlug == "" {
		baseSlug = "workspace"
	}
	slug, err := s.uniqueSlug(ctx, baseSlug, uuid.Nil)
	if err != nil {
		return nil, err
	}

	created, err := s.store.CreateWorkspace(ctx, name, slug, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrSlugConflict) {
			return nil, fmt.Errorf("%w: slug conflict", ErrInvalidInput)
		}
		return nil, err
	}

	channels := make([]domain.Channel, 0, len(created.Channels))
	for _, ch := range created.Channels {
		channels = append(channels, ch.ToDomain())
	}

	result := &CreateWorkspaceResult{
		Workspace: created.Workspace.ToDomain(),
		Channels:  channels,
	}
	if s.lifecycle != nil {
		_ = s.lifecycle.OnWorkspaceCreated(ctx, result.Workspace.ID)
	}
	return result, nil
}

func (s *WorkspaceService) ListForUser(ctx context.Context, userID string) ([]domain.Workspace, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	rows, err := s.store.ListWorkspacesForUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Workspace, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ToDomain())
	}
	return out, nil
}

func (s *WorkspaceService) Get(ctx context.Context, userID, workspaceID string) (*domain.Workspace, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	row, err := s.store.GetWorkspaceForUser(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	ws := row.ToDomain()
	return &ws, nil
}

func (s *WorkspaceService) Rename(ctx context.Context, userID, workspaceID, name string) (*domain.Workspace, error) {
	name = capitaliseName(strings.TrimSpace(name))
	if name == "" || len(name) > 80 {
		return nil, ErrWorkspaceName
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}

	current, err := s.store.GetWorkspaceForUser(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !canManageWorkspace(domain.WorkspaceRole(current.Role)) {
		return nil, ErrForbidden
	}

	baseSlug := slugify(name)
	if baseSlug == "" {
		baseSlug = "workspace"
	}
	slug, err := s.uniqueSlug(ctx, baseSlug, wid)
	if err != nil {
		return nil, err
	}

	updated, err := s.store.UpdateWorkspace(ctx, wid, name, slug)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, postgres.ErrSlugConflict) {
			return nil, fmt.Errorf("%w: slug conflict", ErrInvalidInput)
		}
		return nil, err
	}
	updated.Role = current.Role
	ws := updated.ToDomain()
	return &ws, nil
}

func (s *WorkspaceService) SetIcon(
	ctx context.Context,
	userID, workspaceID string,
	r io.Reader,
	declaredType string,
) (*domain.Workspace, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	current, err := s.store.GetWorkspaceForUser(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !canManageWorkspace(domain.WorkspaceRole(current.Role)) {
		return nil, ErrForbidden
	}

	limited := io.LimitReader(r, maxWorkspaceIconBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > maxWorkspaceIconBytes {
		return nil, ErrInvalidIcon
	}
	detected := http.DetectContentType(data)
	contentType, ok := allowedWorkspaceIconTypes[detected]
	if !ok {
		if declared := strings.ToLower(strings.TrimSpace(declaredType)); declared != "" {
			contentType, ok = allowedWorkspaceIconTypes[declared]
		}
	}
	if !ok {
		return nil, ErrInvalidIcon
	}

	updated, err := s.store.SetWorkspaceIcon(ctx, wid, contentType, data)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	updated.Role = current.Role
	ws := updated.ToDomain()
	return &ws, nil
}

func (s *WorkspaceService) ClearIcon(
	ctx context.Context,
	userID, workspaceID string,
) (*domain.Workspace, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	current, err := s.store.GetWorkspaceForUser(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !canManageWorkspace(domain.WorkspaceRole(current.Role)) {
		return nil, ErrForbidden
	}
	updated, err := s.store.ClearWorkspaceIcon(ctx, wid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	updated.Role = current.Role
	ws := updated.ToDomain()
	return &ws, nil
}

func (s *WorkspaceService) GetIcon(
	ctx context.Context,
	userID, workspaceID string,
) (*postgres.WorkspaceIcon, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	if _, err := s.store.GetWorkspaceForUser(ctx, wid, uid); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	icon, err := s.store.GetWorkspaceIcon(ctx, wid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &icon, nil
}

func (s *WorkspaceService) Delete(ctx context.Context, userID, workspaceID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return ErrNotFound
	}

	current, err := s.store.GetWorkspaceForUser(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if !canManageWorkspace(domain.WorkspaceRole(current.Role)) {
		return ErrForbidden
	}

	if s.lifecycle != nil {
		_ = s.lifecycle.OnWorkspaceDeleting(ctx, workspaceID)
	}

	if err := s.store.DeleteWorkspace(ctx, wid); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *WorkspaceService) ListChannels(ctx context.Context, userID, workspaceID string) ([]domain.Channel, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.store.ListChannelsForWorkspace(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	out := make([]domain.Channel, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ToDomain())
	}
	return out, nil
}

func (s *WorkspaceService) GetMyMembership(
	ctx context.Context,
	userID, workspaceID string,
) (*domain.WorkspaceMember, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	if _, err := s.store.GetWorkspaceForUser(ctx, wid, uid); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	row, err := s.store.GetWorkspaceMember(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toWorkspaceMember(row), nil
}

func (s *WorkspaceService) ListMembers(
	ctx context.Context,
	userID, workspaceID string,
) ([]domain.WorkspaceMember, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	if _, err := s.store.GetWorkspaceForUser(ctx, wid, uid); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	rows, err := s.store.ListWorkspaceMembers(ctx, wid)
	if err != nil {
		return nil, err
	}
	out := make([]domain.WorkspaceMember, 0, len(rows))
	for _, row := range rows {
		out = append(out, *toWorkspaceMember(row))
	}
	return out, nil
}

func (s *WorkspaceService) UpdateMyHandle(
	ctx context.Context,
	userID, workspaceID, handle string,
) (*domain.WorkspaceMember, error) {
	normalised, err := NormaliseHandle(handle)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	if _, err := s.store.GetWorkspaceForUser(ctx, wid, uid); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	taken, err := s.store.MemberHandleExists(ctx, wid, normalised, uid)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrHandleTaken
	}
	row, err := s.store.UpdateWorkspaceMemberHandle(ctx, wid, uid, normalised)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, postgres.ErrHandleConflict) {
			return nil, ErrHandleTaken
		}
		return nil, err
	}
	return toWorkspaceMember(row), nil
}

func (s *WorkspaceService) UpdateMemberRole(
	ctx context.Context, actorID, workspaceID, memberUserID, role string,
) (*domain.WorkspaceMember, error) {
	actorUID, wid, err := parseActorWorkspace(actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	mid, err := uuid.Parse(memberUserID)
	if err != nil {
		return nil, ErrNotFound
	}
	wsRole := domain.WorkspaceRole(strings.TrimSpace(role))
	if wsRole != domain.WorkspaceRoleAdmin && wsRole != domain.WorkspaceRoleMember && wsRole != domain.WorkspaceRoleOwner {
		return nil, ErrInvalidInput
	}
	actor, err := s.store.GetWorkspaceMember(ctx, wid, actorUID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	target, err := s.store.GetWorkspaceMember(ctx, wid, mid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	actorRole := domain.WorkspaceRole(actor.Role)
	targetRole := domain.WorkspaceRole(target.Role)
	if !canManageWorkspace(actorRole) {
		return nil, ErrForbidden
	}
	if actorRole == domain.WorkspaceRoleAdmin {
		if targetRole == domain.WorkspaceRoleOwner || wsRole == domain.WorkspaceRoleOwner {
			return nil, ErrForbidden
		}
	}
	if targetRole == domain.WorkspaceRoleOwner && wsRole != domain.WorkspaceRoleOwner {
		owners, err := s.store.CountWorkspaceOwners(ctx, wid)
		if err != nil {
			return nil, err
		}
		if owners <= 1 {
			return nil, ErrInvalidInput
		}
	}
	row, err := s.store.UpdateWorkspaceMemberRole(ctx, wid, mid, wsRole)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toWorkspaceMember(row), nil
}

func (s *WorkspaceService) RemoveMember(ctx context.Context, actorID, workspaceID, memberUserID string) error {
	actorUID, wid, err := parseActorWorkspace(actorID, workspaceID)
	if err != nil {
		return err
	}
	mid, err := uuid.Parse(memberUserID)
	if err != nil {
		return ErrNotFound
	}
	if actorUID == mid {
		return ErrInvalidInput
	}
	actor, err := s.store.GetWorkspaceMember(ctx, wid, actorUID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	target, err := s.store.GetWorkspaceMember(ctx, wid, mid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	actorRole := domain.WorkspaceRole(actor.Role)
	targetRole := domain.WorkspaceRole(target.Role)
	if !canManageWorkspace(actorRole) {
		return ErrForbidden
	}
	if actorRole == domain.WorkspaceRoleAdmin && targetRole == domain.WorkspaceRoleOwner {
		return ErrForbidden
	}
	if targetRole == domain.WorkspaceRoleOwner {
		owners, err := s.store.CountWorkspaceOwners(ctx, wid)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return ErrInvalidInput
		}
	}
	if err := s.store.RemoveWorkspaceMember(ctx, wid, mid); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if s.lifecycle != nil {
		_ = s.lifecycle.SyncSeats(ctx, workspaceID)
	}
	return nil
}

func (s *WorkspaceService) Leave(ctx context.Context, userID, workspaceID string) error {
	uid, wid, err := parseActorWorkspace(userID, workspaceID)
	if err != nil {
		return err
	}
	member, err := s.store.GetWorkspaceMember(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if domain.WorkspaceRole(member.Role) == domain.WorkspaceRoleOwner {
		owners, err := s.store.CountWorkspaceOwners(ctx, wid)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return ErrInvalidInput
		}
	}
	if err := s.store.RemoveWorkspaceMember(ctx, wid, uid); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if s.lifecycle != nil {
		_ = s.lifecycle.SyncSeats(ctx, workspaceID)
	}
	return nil
}

func (s *WorkspaceService) TransferOwnership(
	ctx context.Context, actorID, workspaceID, newOwnerUserID string,
) error {
	actorUID, wid, err := parseActorWorkspace(actorID, workspaceID)
	if err != nil {
		return err
	}
	newOwner, err := uuid.Parse(newOwnerUserID)
	if err != nil {
		return ErrInvalidInput
	}
	if actorUID == newOwner {
		return ErrInvalidInput
	}
	actor, err := s.store.GetWorkspaceMember(ctx, wid, actorUID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if domain.WorkspaceRole(actor.Role) != domain.WorkspaceRoleOwner {
		return ErrForbidden
	}
	if _, err := s.store.GetWorkspaceMember(ctx, wid, newOwner); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if err := s.store.TransferWorkspaceOwnership(ctx, wid, actorUID, newOwner); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func parseActorWorkspace(actorID, workspaceID string) (uuid.UUID, uuid.UUID, error) {
	uid, err := uuid.Parse(actorID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrNotFound
	}
	return uid, wid, nil
}

func toWorkspaceMember(row postgres.WorkspaceMemberInfo) *domain.WorkspaceMember {
	former := row.FormerHandles
	if former == nil {
		former = []string{}
	}
	kind := row.Kind
	if kind == "" {
		kind = "member"
	}
	out := &domain.WorkspaceMember{
		UserID:                row.UserID.String(),
		DisplayName:           row.DisplayName,
		Handle:                row.Handle,
		FormerHandles:         former,
		Role:                  domain.WorkspaceRole(row.Role),
		Kind:                  kind,
		IsExternal:            kind == "external",
		HomeWorkspaceName:     row.HomeWorkspaceName,
		HomeServerID:          row.HomeServerID,
		HomeWorkspaceRemoteID: row.HomeWorkspaceRemoteID,
		HomeWorkspaceIconURL:  row.HomeWorkspaceIconURL,
		HomeServerURL:         row.HomeServerURL,
		StatusEmoji:           row.StatusEmoji,
		StatusText:            row.StatusText,
		StatusExpiresAt:       row.StatusExpiresAt,
		HasAvatar:             row.HasAvatar,
		AvatarUpdatedAt:       row.AvatarUpdatedAt,
	}
	if row.HomeWorkspaceID != nil {
		out.HomeWorkspaceID = row.HomeWorkspaceID.String()
	}
	return out
}

func canManageWorkspace(role domain.WorkspaceRole) bool {
	return role == domain.WorkspaceRoleOwner || role == domain.WorkspaceRoleAdmin
}

func (s *WorkspaceService) uniqueSlug(ctx context.Context, base string, excludeID uuid.UUID) (string, error) {
	candidate := base
	for i := 0; i < 50; i++ {
		var (
			exists bool
			err    error
		)
		if excludeID == uuid.Nil {
			exists, err = s.store.SlugExists(ctx, candidate)
		} else {
			exists, err = s.store.SlugExistsExcluding(ctx, candidate, excludeID)
		}
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i+2)
	}
	return "", fmt.Errorf("%w: could not allocate slug", ErrInvalidInput)
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func capitaliseName(name string) string {
	runes := []rune(name)
	if len(runes) == 0 {
		return name
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func slugify(name string) string {
	lower := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return '-'
		}
		return unicode.ToLower(r)
	}, name)
	slug := nonSlug.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 48 {
		slug = strings.Trim(slug[:48], "-")
	}
	return slug
}
