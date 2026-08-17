package service_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/id"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type memWorkspaceStore struct {
	mu         sync.Mutex
	workspaces map[uuid.UUID]postgres.WorkspaceRow
	members    map[uuid.UUID]map[uuid.UUID]string
	handles    map[uuid.UUID]map[uuid.UUID]string
	// aliases[workspaceID][handle] = userID
	aliases  map[uuid.UUID]map[string]uuid.UUID
	icons    map[uuid.UUID]postgres.WorkspaceIcon
	channels map[uuid.UUID][]postgres.ChannelRow
	slugs    map[string]uuid.UUID
	users    map[uuid.UUID]postgres.UserRow
}

func newMemWorkspaceStore() *memWorkspaceStore {
	return &memWorkspaceStore{
		workspaces: map[uuid.UUID]postgres.WorkspaceRow{},
		members:    map[uuid.UUID]map[uuid.UUID]string{},
		handles:    map[uuid.UUID]map[uuid.UUID]string{},
		aliases:    map[uuid.UUID]map[string]uuid.UUID{},
		icons:      map[uuid.UUID]postgres.WorkspaceIcon{},
		channels:   map[uuid.UUID][]postgres.ChannelRow{},
		slugs:      map[string]uuid.UUID{},
		users:      map[uuid.UUID]postgres.UserRow{},
	}
}

func (m *memWorkspaceStore) allocateHandleLocked(workspaceID, userID uuid.UUID, displayName string) string {
	base := postgres.BaseHandle(displayName)
	if m.handles[workspaceID] == nil {
		m.handles[workspaceID] = map[uuid.UUID]string{}
	}
	used := map[string]struct{}{}
	for _, h := range m.handles[workspaceID] {
		used[h] = struct{}{}
	}
	candidate := base
	for i := 0; i < 50; i++ {
		if _, ok := used[candidate]; !ok {
			m.handles[workspaceID][userID] = candidate
			return candidate
		}
		candidate = fmt.Sprintf("%s_%d", base, i+2)
	}
	fallback := fmt.Sprintf("member_%s", userID.String()[:8])
	m.handles[workspaceID][userID] = fallback
	return fallback
}

func (m *memWorkspaceStore) CreateWorkspace(
	_ context.Context,
	name, slug string,
	createdBy uuid.UUID,
) (postgres.CreateWorkspaceResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.slugs[slug]; ok {
		return postgres.CreateWorkspaceResult{}, postgres.ErrSlugConflict
	}
	wsID := id.New()
	ws := postgres.WorkspaceRow{
		ID: wsID, Name: name, Slug: slug, CreatedBy: createdBy,
		Role: string(domain.WorkspaceRoleOwner),
	}
	ch := postgres.ChannelRow{
		ID: id.New(), WorkspaceID: wsID, Name: "general", IsPrivate: false, CreatedBy: createdBy,
	}
	m.workspaces[wsID] = ws
	m.slugs[slug] = wsID
	m.members[wsID] = map[uuid.UUID]string{createdBy: string(domain.WorkspaceRoleOwner)}
	displayName := "owner"
	if u, ok := m.users[createdBy]; ok && u.DisplayName != "" {
		displayName = u.DisplayName
	}
	m.allocateHandleLocked(wsID, createdBy, displayName)
	m.channels[wsID] = []postgres.ChannelRow{ch}
	return postgres.CreateWorkspaceResult{Workspace: ws, Channels: []postgres.ChannelRow{ch}}, nil
}

func (m *memWorkspaceStore) ListWorkspacesForUser(_ context.Context, userID uuid.UUID) ([]postgres.WorkspaceRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.WorkspaceRow
	for id, members := range m.members {
		role, ok := members[userID]
		if !ok {
			continue
		}
		ws := m.workspaces[id]
		ws.Role = role
		out = append(out, ws)
	}
	return out, nil
}

func (m *memWorkspaceStore) GetWorkspaceForUser(
	_ context.Context,
	workspaceID, userID uuid.UUID,
) (postgres.WorkspaceRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	members, ok := m.members[workspaceID]
	if !ok {
		return postgres.WorkspaceRow{}, postgres.ErrNotFound
	}
	role, ok := members[userID]
	if !ok {
		return postgres.WorkspaceRow{}, postgres.ErrNotFound
	}
	ws := m.workspaces[workspaceID]
	ws.Role = role
	return ws, nil
}

func (m *memWorkspaceStore) SlugExists(_ context.Context, slug string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.slugs[slug]
	return ok, nil
}

func (m *memWorkspaceStore) SlugExistsExcluding(_ context.Context, slug string, excludeID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.slugs[slug]
	if !ok {
		return false, nil
	}
	return id != excludeID, nil
}

func (m *memWorkspaceStore) UpdateWorkspace(
	_ context.Context,
	workspaceID uuid.UUID,
	name, slug string,
) (postgres.WorkspaceRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ws, ok := m.workspaces[workspaceID]
	if !ok {
		return postgres.WorkspaceRow{}, postgres.ErrNotFound
	}
	if other, taken := m.slugs[slug]; taken && other != workspaceID {
		return postgres.WorkspaceRow{}, postgres.ErrSlugConflict
	}
	delete(m.slugs, ws.Slug)
	ws.Name = name
	ws.Slug = slug
	m.workspaces[workspaceID] = ws
	m.slugs[slug] = workspaceID
	return ws, nil
}

func (m *memWorkspaceStore) SetWorkspaceIcon(
	_ context.Context, workspaceID uuid.UUID, contentType string, data []byte,
) (postgres.WorkspaceRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ws, ok := m.workspaces[workspaceID]
	if !ok {
		return postgres.WorkspaceRow{}, postgres.ErrNotFound
	}
	now := time.Now().UTC()
	m.icons[workspaceID] = postgres.WorkspaceIcon{
		ContentType: contentType, Bytes: append([]byte(nil), data...), UpdatedAt: now,
	}
	ws.HasIcon = true
	ws.IconContentType = contentType
	ws.IconUpdatedAt = &now
	m.workspaces[workspaceID] = ws
	return ws, nil
}

func (m *memWorkspaceStore) ClearWorkspaceIcon(
	_ context.Context, workspaceID uuid.UUID,
) (postgres.WorkspaceRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ws, ok := m.workspaces[workspaceID]
	if !ok {
		return postgres.WorkspaceRow{}, postgres.ErrNotFound
	}
	delete(m.icons, workspaceID)
	ws.HasIcon = false
	ws.IconContentType = ""
	ws.IconUpdatedAt = nil
	m.workspaces[workspaceID] = ws
	return ws, nil
}

func (m *memWorkspaceStore) GetWorkspaceIcon(
	_ context.Context, workspaceID uuid.UUID,
) (postgres.WorkspaceIcon, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	icon, ok := m.icons[workspaceID]
	if !ok {
		return postgres.WorkspaceIcon{}, postgres.ErrNotFound
	}
	return icon, nil
}

func (m *memWorkspaceStore) DeleteWorkspace(_ context.Context, workspaceID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ws, ok := m.workspaces[workspaceID]
	if !ok {
		return postgres.ErrNotFound
	}
	delete(m.slugs, ws.Slug)
	delete(m.workspaces, workspaceID)
	delete(m.members, workspaceID)
	delete(m.handles, workspaceID)
	delete(m.icons, workspaceID)
	delete(m.channels, workspaceID)
	return nil
}

func (m *memWorkspaceStore) formerHandlesLocked(workspaceID, userID uuid.UUID) []string {
	var out []string
	for handle, owner := range m.aliases[workspaceID] {
		if owner == userID {
			out = append(out, handle)
		}
	}
	return out
}

func (m *memWorkspaceStore) GetWorkspaceMember(
	_ context.Context, workspaceID, userID uuid.UUID,
) (postgres.WorkspaceMemberInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	role, ok := m.members[workspaceID][userID]
	if !ok {
		return postgres.WorkspaceMemberInfo{}, postgres.ErrNotFound
	}
	handle := m.handles[workspaceID][userID]
	name := ""
	if u, ok := m.users[userID]; ok {
		name = u.DisplayName
	}
	return postgres.WorkspaceMemberInfo{
		UserID: userID, DisplayName: name, Handle: handle,
		FormerHandles: m.formerHandlesLocked(workspaceID, userID), Role: role,
	}, nil
}

func (m *memWorkspaceStore) ListWorkspaceMembers(
	_ context.Context, workspaceID uuid.UUID,
) ([]postgres.WorkspaceMemberInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.WorkspaceMemberInfo
	for userID, role := range m.members[workspaceID] {
		name := ""
		if u, ok := m.users[userID]; ok {
			name = u.DisplayName
		}
		out = append(out, postgres.WorkspaceMemberInfo{
			UserID: userID, DisplayName: name, Handle: m.handles[workspaceID][userID],
			FormerHandles: m.formerHandlesLocked(workspaceID, userID), Role: role,
		})
	}
	return out, nil
}

func (m *memWorkspaceStore) MemberHandleExists(
	_ context.Context, workspaceID uuid.UUID, handle string, excludeUserID uuid.UUID,
) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for userID, h := range m.handles[workspaceID] {
		if userID != excludeUserID && h == handle {
			return true, nil
		}
	}
	return false, nil
}

func (m *memWorkspaceStore) UpdateWorkspaceMemberHandle(
	_ context.Context, workspaceID, userID uuid.UUID, handle string,
) (postgres.WorkspaceMemberInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	role, ok := m.members[workspaceID][userID]
	if !ok {
		return postgres.WorkspaceMemberInfo{}, postgres.ErrNotFound
	}
	for otherID, h := range m.handles[workspaceID] {
		if otherID != userID && h == handle {
			return postgres.WorkspaceMemberInfo{}, postgres.ErrHandleConflict
		}
	}
	if m.handles[workspaceID] == nil {
		m.handles[workspaceID] = map[uuid.UUID]string{}
	}
	if m.aliases[workspaceID] == nil {
		m.aliases[workspaceID] = map[string]uuid.UUID{}
	}
	oldHandle := m.handles[workspaceID][userID]
	if oldHandle != handle {
		delete(m.aliases[workspaceID], handle)
		m.aliases[workspaceID][oldHandle] = userID
		m.handles[workspaceID][userID] = handle
	}
	name := ""
	if u, ok := m.users[userID]; ok {
		name = u.DisplayName
	}
	return postgres.WorkspaceMemberInfo{
		UserID: userID, DisplayName: name, Handle: handle,
		FormerHandles: m.formerHandlesLocked(workspaceID, userID), Role: role,
	}, nil
}

func (m *memWorkspaceStore) ListChannelsForWorkspace(
	_ context.Context,
	workspaceID, userID uuid.UUID,
) ([]postgres.ChannelRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	members, ok := m.members[workspaceID]
	if !ok {
		return nil, postgres.ErrNotFound
	}
	if _, ok := members[userID]; !ok {
		return nil, postgres.ErrNotFound
	}
	var out []postgres.ChannelRow
	for _, ch := range m.channels[workspaceID] {
		if !ch.IsPrivate {
			out = append(out, ch)
		}
	}
	return out, nil
}

func TestWorkspaceCreateSeedsGeneral(t *testing.T) {
	t.Parallel()
	svc := service.NewWorkspaceService(newMemWorkspaceStore())
	userID := id.New().String()

	created, err := svc.Create(context.Background(), userID, "kilobyte")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Workspace.Name != "Kilobyte" {
		t.Fatalf("name = %q, want Kilobyte", created.Workspace.Name)
	}
	if created.Workspace.Slug != "kilobyte" {
		t.Fatalf("slug = %q", created.Workspace.Slug)
	}
	if created.Workspace.Role != domain.WorkspaceRoleOwner {
		t.Fatalf("role = %q", created.Workspace.Role)
	}
	if len(created.Channels) != 1 || created.Channels[0].Name != "general" {
		t.Fatalf("channels = %+v", created.Channels)
	}

	list, err := svc.ListForUser(context.Background(), userID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %#v err=%v", list, err)
	}

	other := id.New().String()
	otherList, err := svc.ListForUser(context.Background(), other)
	if err != nil || len(otherList) != 0 {
		t.Fatalf("other list should be empty: %#v err=%v", otherList, err)
	}

	_, err = svc.ListChannels(context.Background(), other, created.Workspace.ID)
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected not found for non-member, got %v", err)
	}

	channels, err := svc.ListChannels(context.Background(), userID, created.Workspace.ID)
	if err != nil || len(channels) != 1 {
		t.Fatalf("channels: %#v err=%v", channels, err)
	}
}

func TestWorkspaceSlugUniqueness(t *testing.T) {
	t.Parallel()
	svc := service.NewWorkspaceService(newMemWorkspaceStore())
	userID := id.New().String()
	if _, err := svc.Create(context.Background(), userID, "Acme"); err != nil {
		t.Fatal(err)
	}
	second, err := svc.Create(context.Background(), userID, "Acme")
	if err != nil {
		t.Fatal(err)
	}
	if second.Workspace.Slug != "acme-2" {
		t.Fatalf("slug = %q, want acme-2", second.Workspace.Slug)
	}
}

func TestWorkspaceRenameAndDelete(t *testing.T) {
	t.Parallel()
	svc := service.NewWorkspaceService(newMemWorkspaceStore())
	userID := id.New().String()
	created, err := svc.Create(context.Background(), userID, "Acme")
	if err != nil {
		t.Fatal(err)
	}

	renamed, err := svc.Rename(context.Background(), userID, created.Workspace.ID, "northwind")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed.Name != "Northwind" || renamed.Slug != "northwind" {
		t.Fatalf("renamed = %+v", renamed)
	}

	other := id.New().String()
	_, err = svc.Rename(context.Background(), other, created.Workspace.ID, "Nope")
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("non-member rename: %v", err)
	}

	if err := svc.Delete(context.Background(), userID, created.Workspace.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, err := svc.ListForUser(context.Background(), userID)
	if err != nil || len(list) != 0 {
		t.Fatalf("list after delete: %#v err=%v", list, err)
	}
}

func TestWorkspaceHandleUnique(t *testing.T) {
	t.Parallel()
	store := newMemWorkspaceStore()
	ownerID := id.New()
	memberID := id.New()
	store.users[ownerID] = postgres.UserRow{ID: ownerID, DisplayName: "Ada Lovelace"}
	store.users[memberID] = postgres.UserRow{ID: memberID, DisplayName: "Other"}
	svc := service.NewWorkspaceService(store)

	created, err := svc.Create(context.Background(), ownerID.String(), "Lab")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	wid := uuid.MustParse(created.Workspace.ID)
	store.mu.Lock()
	store.members[wid][memberID] = "member"
	store.allocateHandleLocked(wid, memberID, "Other")
	store.mu.Unlock()

	me, err := svc.GetMyMembership(context.Background(), ownerID.String(), created.Workspace.ID)
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	if me.Handle != "ada_lovelace" {
		t.Fatalf("handle = %q", me.Handle)
	}

	updated, err := svc.UpdateMyHandle(context.Background(), ownerID.String(), created.Workspace.ID, "ada")
	if err != nil {
		t.Fatalf("update handle: %v", err)
	}
	if updated.Handle != "ada" {
		t.Fatalf("updated = %q", updated.Handle)
	}
	if len(updated.FormerHandles) != 1 || updated.FormerHandles[0] != "ada_lovelace" {
		t.Fatalf("former handles = %#v", updated.FormerHandles)
	}

	if _, err := svc.UpdateMyHandle(context.Background(), memberID.String(), created.Workspace.ID, "ada"); !errors.Is(err, service.ErrHandleTaken) {
		t.Fatalf("expected handle taken, got %v", err)
	}
	if _, err := svc.UpdateMyHandle(context.Background(), ownerID.String(), created.Workspace.ID, "x"); !errors.Is(err, service.ErrInvalidHandle) {
		t.Fatalf("expected invalid handle, got %v", err)
	}

	claimed, err := svc.UpdateMyHandle(context.Background(), memberID.String(), created.Workspace.ID, "ada_lovelace")
	if err != nil {
		t.Fatalf("reclaim former handle: %v", err)
	}
	if claimed.Handle != "ada_lovelace" {
		t.Fatalf("claimed = %q", claimed.Handle)
	}
	ownerAgain, err := svc.GetMyMembership(context.Background(), ownerID.String(), created.Workspace.ID)
	if err != nil {
		t.Fatalf("owner again: %v", err)
	}
	for _, former := range ownerAgain.FormerHandles {
		if former == "ada_lovelace" {
			t.Fatalf("reclaimed alias should leave owner: %#v", ownerAgain.FormerHandles)
		}
	}
}

func (m *memWorkspaceStore) CountWorkspaceOwners(_ context.Context, workspaceID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, role := range m.members[workspaceID] {
		if role == string(domain.WorkspaceRoleOwner) {
			n++
		}
	}
	return n, nil
}

func (m *memWorkspaceStore) UpdateWorkspaceMemberRole(
	_ context.Context, workspaceID, userID uuid.UUID, role domain.WorkspaceRole,
) (postgres.WorkspaceMemberInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.members[workspaceID][userID]; !ok {
		return postgres.WorkspaceMemberInfo{}, postgres.ErrNotFound
	}
	m.members[workspaceID][userID] = string(role)
	name := ""
	if u, ok := m.users[userID]; ok {
		name = u.DisplayName
	}
	return postgres.WorkspaceMemberInfo{
		UserID: userID, DisplayName: name, Handle: m.handles[workspaceID][userID],
		FormerHandles: m.formerHandlesLocked(workspaceID, userID), Role: string(role), Kind: "member",
	}, nil
}

func (m *memWorkspaceStore) RemoveWorkspaceMember(_ context.Context, workspaceID, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.members[workspaceID][userID]; !ok {
		return postgres.ErrNotFound
	}
	delete(m.members[workspaceID], userID)
	delete(m.handles[workspaceID], userID)
	return nil
}

func (m *memWorkspaceStore) TransferWorkspaceOwnership(
	_ context.Context, workspaceID, fromOwner, toUser uuid.UUID,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members[workspaceID][fromOwner] != string(domain.WorkspaceRoleOwner) {
		return postgres.ErrNotFound
	}
	if _, ok := m.members[workspaceID][toUser]; !ok {
		return postgres.ErrNotFound
	}
	m.members[workspaceID][fromOwner] = string(domain.WorkspaceRoleMember)
	m.members[workspaceID][toUser] = string(domain.WorkspaceRoleOwner)
	return nil
}

func TestWorkspacePeopleManagement(t *testing.T) {
	t.Parallel()
	store := newMemWorkspaceStore()
	ownerID := id.New()
	adminID := id.New()
	memberID := id.New()
	store.users[ownerID] = postgres.UserRow{ID: ownerID, DisplayName: "Owner"}
	store.users[adminID] = postgres.UserRow{ID: adminID, DisplayName: "Admin"}
	store.users[memberID] = postgres.UserRow{ID: memberID, DisplayName: "Member"}

	svc := service.NewWorkspaceService(store)
	created, err := svc.Create(context.Background(), ownerID.String(), "People Lab")
	if err != nil {
		t.Fatal(err)
	}
	wsID := created.Workspace.ID
	wid, _ := uuid.Parse(wsID)
	_ = store.members[wid]
	store.members[wid][adminID] = string(domain.WorkspaceRoleAdmin)
	store.members[wid][memberID] = string(domain.WorkspaceRoleMember)
	if store.handles[wid] == nil {
		store.handles[wid] = map[uuid.UUID]string{}
	}
	store.handles[wid][adminID] = "admin"
	store.handles[wid][memberID] = "member"

	updated, err := svc.UpdateMemberRole(context.Background(), ownerID.String(), wsID, memberID.String(), "admin")
	if err != nil || updated.Role != domain.WorkspaceRoleAdmin {
		t.Fatalf("promote = %+v err=%v", updated, err)
	}

	if err := svc.TransferOwnership(context.Background(), ownerID.String(), wsID, adminID.String()); err != nil {
		t.Fatal(err)
	}
	if store.members[wid][adminID] != string(domain.WorkspaceRoleOwner) {
		t.Fatalf("expected admin to be owner, got %s", store.members[wid][adminID])
	}

	if err := svc.RemoveMember(context.Background(), adminID.String(), wsID, memberID.String()); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.members[wid][memberID]; ok {
		t.Fatal("member should be removed")
	}

	if err := svc.Leave(context.Background(), ownerID.String(), wsID); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.members[wid][ownerID]; ok {
		t.Fatal("former owner should have left")
	}

	if err := svc.Leave(context.Background(), adminID.String(), wsID); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("sole owner leave = %v", err)
	}
}
