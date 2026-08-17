package service_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/auth"
	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/id"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type memDomainHarness struct {
	mu         sync.Mutex
	workspaces map[uuid.UUID]postgres.WorkspaceRow
	members    map[uuid.UUID]map[uuid.UUID]string
	handles    map[uuid.UUID]map[uuid.UUID]string
	domains    map[uuid.UUID]postgres.WorkspaceDomainRow
	users      *memStore
}

func newMemDomainHarness() *memDomainHarness {
	return &memDomainHarness{
		workspaces: map[uuid.UUID]postgres.WorkspaceRow{},
		members:    map[uuid.UUID]map[uuid.UUID]string{},
		handles:    map[uuid.UUID]map[uuid.UUID]string{},
		domains:    map[uuid.UUID]postgres.WorkspaceDomainRow{},
		users:      newMemStore(),
	}
}

func (m *memDomainHarness) allocateHandleLocked(workspaceID, userID uuid.UUID, displayName string) string {
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

func (m *memDomainHarness) CreateWorkspace(
	_ context.Context, name, slug string, createdBy uuid.UUID,
) (postgres.CreateWorkspaceResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	wsID := id.New()
	ws := postgres.WorkspaceRow{
		ID: wsID, Name: name, Slug: slug, CreatedBy: createdBy,
		Role: string(domain.WorkspaceRoleOwner),
	}
	m.workspaces[wsID] = ws
	m.members[wsID] = map[uuid.UUID]string{createdBy: string(domain.WorkspaceRoleOwner)}
	displayName := "owner"
	if u, ok := m.users.byID[createdBy]; ok && u.DisplayName != "" {
		displayName = u.DisplayName
	}
	m.allocateHandleLocked(wsID, createdBy, displayName)
	return postgres.CreateWorkspaceResult{Workspace: ws}, nil
}

func (m *memDomainHarness) ListWorkspacesForUser(_ context.Context, userID uuid.UUID) ([]postgres.WorkspaceRow, error) {
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

func (m *memDomainHarness) GetWorkspaceForUser(
	_ context.Context, workspaceID, userID uuid.UUID,
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

func (m *memDomainHarness) SlugExists(context.Context, string) (bool, error) { return false, nil }
func (m *memDomainHarness) SlugExistsExcluding(context.Context, string, uuid.UUID) (bool, error) {
	return false, nil
}
func (m *memDomainHarness) UpdateWorkspace(context.Context, uuid.UUID, string, string) (postgres.WorkspaceRow, error) {
	return postgres.WorkspaceRow{}, postgres.ErrNotFound
}
func (m *memDomainHarness) DeleteWorkspace(context.Context, uuid.UUID) error {
	return postgres.ErrNotFound
}
func (m *memDomainHarness) ListChannelsForWorkspace(context.Context, uuid.UUID, uuid.UUID) ([]postgres.ChannelRow, error) {
	return nil, nil
}
func (m *memDomainHarness) GetWorkspaceMember(context.Context, uuid.UUID, uuid.UUID) (postgres.WorkspaceMemberInfo, error) {
	return postgres.WorkspaceMemberInfo{}, postgres.ErrNotFound
}
func (m *memDomainHarness) ListWorkspaceMembers(context.Context, uuid.UUID) ([]postgres.WorkspaceMemberInfo, error) {
	return nil, nil
}
func (m *memDomainHarness) MemberHandleExists(context.Context, uuid.UUID, string, uuid.UUID) (bool, error) {
	return false, nil
}
func (m *memDomainHarness) UpdateWorkspaceMemberHandle(context.Context, uuid.UUID, uuid.UUID, string) (postgres.WorkspaceMemberInfo, error) {
	return postgres.WorkspaceMemberInfo{}, postgres.ErrNotFound
}
func (m *memDomainHarness) SetWorkspaceIcon(context.Context, uuid.UUID, string, []byte) (postgres.WorkspaceRow, error) {
	return postgres.WorkspaceRow{}, postgres.ErrNotFound
}
func (m *memDomainHarness) ClearWorkspaceIcon(context.Context, uuid.UUID) (postgres.WorkspaceRow, error) {
	return postgres.WorkspaceRow{}, postgres.ErrNotFound
}
func (m *memDomainHarness) GetWorkspaceIcon(context.Context, uuid.UUID) (postgres.WorkspaceIcon, error) {
	return postgres.WorkspaceIcon{}, postgres.ErrNotFound
}

func (m *memDomainHarness) CreateWorkspaceDomain(
	_ context.Context, workspaceID, createdBy uuid.UUID, domainName, verificationToken string,
) (postgres.WorkspaceDomainRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, d := range m.domains {
		if d.WorkspaceID == workspaceID && d.Domain == domainName {
			return postgres.WorkspaceDomainRow{}, postgres.ErrDomainConflict
		}
	}
	now := time.Now().UTC()
	row := postgres.WorkspaceDomainRow{
		ID: id.New(), WorkspaceID: workspaceID, Domain: domainName,
		VerificationToken: verificationToken, CreatedBy: createdBy,
		CreatedAt: now, UpdatedAt: now,
	}
	m.domains[row.ID] = row
	return row, nil
}

func (m *memDomainHarness) ListWorkspaceDomains(_ context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceDomainRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.WorkspaceDomainRow
	for _, d := range m.domains {
		if d.WorkspaceID == workspaceID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (m *memDomainHarness) GetWorkspaceDomain(
	_ context.Context, workspaceID, domainID uuid.UUID,
) (postgres.WorkspaceDomainRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.domains[domainID]
	if !ok || row.WorkspaceID != workspaceID {
		return postgres.WorkspaceDomainRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *memDomainHarness) MarkWorkspaceDomainVerified(
	_ context.Context, workspaceID, domainID uuid.UUID,
) (postgres.WorkspaceDomainRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.domains[domainID]
	if !ok || row.WorkspaceID != workspaceID {
		return postgres.WorkspaceDomainRow{}, postgres.ErrNotFound
	}
	for _, d := range m.domains {
		if d.Domain == row.Domain && d.VerifiedAt != nil && d.WorkspaceID != workspaceID {
			return postgres.WorkspaceDomainRow{}, postgres.ErrDomainVerifiedConflict
		}
	}
	now := time.Now().UTC()
	row.VerifiedAt = &now
	row.UpdatedAt = now
	m.domains[domainID] = row
	return row, nil
}

func (m *memDomainHarness) UpdateWorkspaceDomainAutoJoin(
	_ context.Context, workspaceID, domainID uuid.UUID, autoJoin bool,
) (postgres.WorkspaceDomainRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.domains[domainID]
	if !ok || row.WorkspaceID != workspaceID || row.VerifiedAt == nil {
		return postgres.WorkspaceDomainRow{}, postgres.ErrNotFound
	}
	row.AutoJoin = autoJoin
	row.UpdatedAt = time.Now().UTC()
	m.domains[domainID] = row
	return row, nil
}

func (m *memDomainHarness) DeleteWorkspaceDomain(_ context.Context, workspaceID, domainID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.domains[domainID]
	if !ok || row.WorkspaceID != workspaceID {
		return postgres.ErrNotFound
	}
	delete(m.domains, domainID)
	return nil
}

func (m *memDomainHarness) DomainVerifiedElsewhere(
	_ context.Context, domainName string, excludeWorkspaceID uuid.UUID,
) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, d := range m.domains {
		if d.Domain == domainName && d.VerifiedAt != nil && d.WorkspaceID != excludeWorkspaceID {
			return true, nil
		}
	}
	return false, nil
}

func (m *memDomainHarness) ListAutoJoinWorkspaceIDsByDomain(_ context.Context, domainName string) ([]uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []uuid.UUID
	for _, d := range m.domains {
		if d.Domain == domainName && d.VerifiedAt != nil && d.AutoJoin {
			out = append(out, d.WorkspaceID)
		}
	}
	return out, nil
}

func (m *memDomainHarness) AddWorkspaceMember(
	_ context.Context, workspaceID, userID uuid.UUID, role domain.WorkspaceRole,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members[workspaceID] == nil {
		m.members[workspaceID] = map[uuid.UUID]string{}
	}
	if _, ok := m.members[workspaceID][userID]; !ok {
		m.members[workspaceID][userID] = string(role)
		displayName := "member"
		if u, ok := m.users.byID[userID]; ok && u.DisplayName != "" {
			displayName = u.DisplayName
		}
		m.allocateHandleLocked(workspaceID, userID, displayName)
	}
	return nil
}

func TestDomainAddVerifyAutoJoinSignup(t *testing.T) {
	t.Parallel()
	h := newMemDomainHarness()
	domainSvc := service.NewDomainService(h)
	wsSvc := service.NewWorkspaceService(h)
	authSvc := service.NewAuthService(h.users, auth.PasswordPolicy{
		MinLength: 12, RequireUppercase: true, RequireLowercase: true, RequireNumber: true,
	}, time.Hour).WithAutoJoiner(domainSvc)

	owner, err := authSvc.Signup(context.Background(), service.SignupInput{
		Email: "owner@acme.test", Password: "ValidPass1234", DisplayName: "Owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	created, err := wsSvc.Create(context.Background(), owner.User.ID, "Acme")
	if err != nil {
		t.Fatal(err)
	}

	_, err = domainSvc.Add(context.Background(), owner.User.ID, created.Workspace.ID, "gmail.com")
	if !errors.Is(err, service.ErrDomainDenied) {
		t.Fatalf("expected denied, got %v", err)
	}

	pending, err := domainSvc.Add(context.Background(), owner.User.ID, created.Workspace.ID, "acme.test")
	if err != nil {
		t.Fatal(err)
	}

	domainSvc.WithTXTResolver(func(host string) ([]string, error) {
		if host != service.DomainTXTHost("acme.test") {
			t.Fatalf("host = %q", host)
		}
		return []string{"unrelated", service.DomainTXTValue(pending.VerificationToken)}, nil
	})

	verified, err := domainSvc.Verify(context.Background(), owner.User.ID, created.Workspace.ID, pending.ID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !verified.Verified() {
		t.Fatal("expected verified")
	}

	_, err = domainSvc.SetAutoJoin(context.Background(), owner.User.ID, created.Workspace.ID, pending.ID, true)
	if err != nil {
		t.Fatal(err)
	}

	member, err := authSvc.Signup(context.Background(), service.SignupInput{
		Email: "dev@acme.test", Password: "ValidPass1234", DisplayName: "Dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	list, err := wsSvc.ListForUser(context.Background(), member.User.ID)
	if err != nil || len(list) != 1 || list[0].ID != created.Workspace.ID {
		t.Fatalf("auto-join list: %#v err=%v", list, err)
	}
	if list[0].Role != domain.WorkspaceRoleMember {
		t.Fatalf("role = %q", list[0].Role)
	}
}

func TestDomainVerifyDNSMismatch(t *testing.T) {
	t.Parallel()
	h := newMemDomainHarness()
	domainSvc := service.NewDomainService(h)
	wsSvc := service.NewWorkspaceService(h)
	authSvc := service.NewAuthService(h.users, auth.PasswordPolicy{
		MinLength: 12, RequireUppercase: true, RequireLowercase: true, RequireNumber: true,
	}, time.Hour)

	owner, err := authSvc.Signup(context.Background(), service.SignupInput{
		Email: "owner@corp.example", Password: "ValidPass1234", DisplayName: "Owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	created, err := wsSvc.Create(context.Background(), owner.User.ID, "Corp")
	if err != nil {
		t.Fatal(err)
	}
	pending, err := domainSvc.Add(context.Background(), owner.User.ID, created.Workspace.ID, "corp.example")
	if err != nil {
		t.Fatal(err)
	}
	domainSvc.WithTXTResolver(func(string) ([]string, error) {
		return []string{"wrong-value"}, nil
	})
	_, err = domainSvc.Verify(context.Background(), owner.User.ID, created.Workspace.ID, pending.ID)
	if !errors.Is(err, service.ErrDomainDNSMismatch) {
		t.Fatalf("expected dns mismatch, got %v", err)
	}
}

func TestDomainNoAutoJoinWhenDisabled(t *testing.T) {
	t.Parallel()
	h := newMemDomainHarness()
	domainSvc := service.NewDomainService(h)
	wsSvc := service.NewWorkspaceService(h)
	authSvc := service.NewAuthService(h.users, auth.PasswordPolicy{
		MinLength: 12, RequireUppercase: true, RequireLowercase: true, RequireNumber: true,
	}, time.Hour).WithAutoJoiner(domainSvc)

	owner, err := authSvc.Signup(context.Background(), service.SignupInput{
		Email: "owner@fabrikam.example", Password: "ValidPass1234", DisplayName: "Owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	created, err := wsSvc.Create(context.Background(), owner.User.ID, "Fabrikam")
	if err != nil {
		t.Fatal(err)
	}
	pending, err := domainSvc.Add(context.Background(), owner.User.ID, created.Workspace.ID, "fabrikam.example")
	if err != nil {
		t.Fatal(err)
	}
	domainSvc.WithTXTResolver(func(string) ([]string, error) {
		return []string{service.DomainTXTValue(pending.VerificationToken)}, nil
	})
	if _, err := domainSvc.Verify(context.Background(), owner.User.ID, created.Workspace.ID, pending.ID); err != nil {
		t.Fatal(err)
	}

	newbie, err := authSvc.Signup(context.Background(), service.SignupInput{
		Email: "new@fabrikam.example", Password: "ValidPass1234", DisplayName: "New",
	})
	if err != nil {
		t.Fatal(err)
	}
	list, err := wsSvc.ListForUser(context.Background(), newbie.User.ID)
	if err != nil || len(list) != 0 {
		t.Fatalf("expected no auto-join, got %#v err=%v", list, err)
	}
}

func (m *memDomainHarness) CountWorkspaceOwners(context.Context, uuid.UUID) (int, error) { return 0, nil }
func (m *memDomainHarness) UpdateWorkspaceMemberRole(context.Context, uuid.UUID, uuid.UUID, domain.WorkspaceRole) (postgres.WorkspaceMemberInfo, error) {
	return postgres.WorkspaceMemberInfo{}, postgres.ErrNotFound
}
func (m *memDomainHarness) RemoveWorkspaceMember(context.Context, uuid.UUID, uuid.UUID) error {
	return postgres.ErrNotFound
}
func (m *memDomainHarness) TransferWorkspaceOwnership(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return postgres.ErrNotFound
}
