package service_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/id"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type documentMemStore struct {
	mu         sync.Mutex
	workspaces map[uuid.UUID]postgres.WorkspaceRow
	members    map[uuid.UUID]map[uuid.UUID]string
	users      map[uuid.UUID]postgres.UserRow
	docs       map[uuid.UUID]postgres.DocumentRow
	revisions  map[uuid.UUID]postgres.DocumentRevisionRow
}

func newDocumentMemStore() *documentMemStore {
	return &documentMemStore{
		workspaces: map[uuid.UUID]postgres.WorkspaceRow{},
		members:    map[uuid.UUID]map[uuid.UUID]string{},
		users:      map[uuid.UUID]postgres.UserRow{},
		docs:       map[uuid.UUID]postgres.DocumentRow{},
		revisions:  map[uuid.UUID]postgres.DocumentRevisionRow{},
	}
}

func (m *documentMemStore) seedUser() postgres.UserRow {
	now := time.Now().UTC()
	user := postgres.UserRow{ID: id.New(), Email: "owner@example.com", DisplayName: "Owner", CreatedAt: now, UpdatedAt: now}
	m.users[user.ID] = user
	return user
}

func (m *documentMemStore) seedWorkspace(owner uuid.UUID) postgres.WorkspaceRow {
	m.mu.Lock()
	defer m.mu.Unlock()
	ws := postgres.WorkspaceRow{
		ID: id.New(), Name: "Acme", Slug: "acme", CreatedBy: owner, Role: "owner",
	}
	m.workspaces[ws.ID] = ws
	m.members[ws.ID] = map[uuid.UUID]string{owner: "owner"}
	return ws
}

func (m *documentMemStore) addMember(workspaceID, userID uuid.UUID, role string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members[workspaceID] == nil {
		m.members[workspaceID] = map[uuid.UUID]string{}
	}
	m.members[workspaceID][userID] = role
}

func (m *documentMemStore) GetUserByID(_ context.Context, userID uuid.UUID) (postgres.UserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.users[userID]
	if !ok {
		return postgres.UserRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *documentMemStore) GetWorkspaceForUser(_ context.Context, workspaceID, userID uuid.UUID) (postgres.WorkspaceRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	role, ok := m.members[workspaceID][userID]
	if !ok {
		return postgres.WorkspaceRow{}, postgres.ErrNotFound
	}
	ws := m.workspaces[workspaceID]
	ws.Role = role
	return ws, nil
}

func (m *documentMemStore) IsWorkspaceMember(_ context.Context, workspaceID, userID uuid.UUID) (bool, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	role, ok := m.members[workspaceID][userID]
	if !ok {
		return false, "", nil
	}
	return true, role, nil
}

func (m *documentMemStore) InsertDocument(
	_ context.Context, workspaceID uuid.UUID, parentID *uuid.UUID, slug, title, body, icon string, authorID uuid.UUID,
) (postgres.DocumentRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, row := range m.docs {
		if row.WorkspaceID == workspaceID && row.Slug == slug {
			return postgres.DocumentRow{}, postgres.ErrDocumentSlugConflict
		}
	}
	now := time.Now().UTC()
	row := postgres.DocumentRow{
		ID: id.New(), WorkspaceID: workspaceID, ParentID: parentID,
		Slug: slug, Title: title, Body: body, Icon: icon,
		CreatedBy: authorID, UpdatedBy: authorID,
		CreatedAt: now, UpdatedAt: now,
	}
	m.docs[row.ID] = row
	m.addRevisionLocked(row)
	return row, nil
}

func (m *documentMemStore) GetDocument(_ context.Context, documentID uuid.UUID) (postgres.DocumentRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.docs[documentID]
	if !ok {
		return postgres.DocumentRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *documentMemStore) ListDocuments(_ context.Context, workspaceID uuid.UUID) ([]postgres.DocumentRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.DocumentRow
	for _, row := range m.docs {
		if row.WorkspaceID == workspaceID {
			copy := row
			copy.Body = ""
			out = append(out, copy)
		}
	}
	return out, nil
}

func (m *documentMemStore) SearchDocuments(_ context.Context, workspaceID uuid.UUID, query string, limit int) ([]postgres.DocumentRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	q := strings.ToLower(query)
	var out []postgres.DocumentRow
	for _, row := range m.docs {
		if row.WorkspaceID != workspaceID {
			continue
		}
		if strings.Contains(strings.ToLower(row.Title), q) || strings.Contains(strings.ToLower(row.Slug), q) {
			copy := row
			copy.Body = ""
			out = append(out, copy)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (m *documentMemStore) DocumentSlugExists(_ context.Context, workspaceID uuid.UUID, slug string, exceptID *uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, row := range m.docs {
		if row.WorkspaceID != workspaceID || row.Slug != slug {
			continue
		}
		if exceptID != nil && row.ID == *exceptID {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (m *documentMemStore) CountDocumentChildren(_ context.Context, documentID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, row := range m.docs {
		if row.ParentID != nil && *row.ParentID == documentID {
			count++
		}
	}
	return count, nil
}

func (m *documentMemStore) UpdateDocument(
	_ context.Context, documentID uuid.UUID, parentID *uuid.UUID, slug, title, body, icon string, updatedBy uuid.UUID,
) (postgres.DocumentRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.docs[documentID]
	if !ok {
		return postgres.DocumentRow{}, postgres.ErrNotFound
	}
	for _, other := range m.docs {
		if other.WorkspaceID == row.WorkspaceID && other.Slug == slug && other.ID != documentID {
			return postgres.DocumentRow{}, postgres.ErrDocumentSlugConflict
		}
	}
	row.ParentID = parentID
	row.Slug = slug
	row.Title = title
	row.Body = body
	row.Icon = icon
	row.UpdatedBy = updatedBy
	row.UpdatedAt = time.Now().UTC()
	m.docs[documentID] = row
	m.addRevisionLocked(row)
	return row, nil
}

func (m *documentMemStore) addRevisionLocked(doc postgres.DocumentRow) {
	version := 1
	for _, rev := range m.revisions {
		if rev.DocumentID == doc.ID && rev.Version >= version {
			version = rev.Version + 1
		}
	}
	row := postgres.DocumentRevisionRow{
		ID: id.New(), DocumentID: doc.ID, Version: version,
		ParentID: doc.ParentID, Slug: doc.Slug, Title: doc.Title,
		Body: doc.Body, Icon: doc.Icon, CreatedBy: doc.UpdatedBy,
		CreatedAt: time.Now().UTC(),
	}
	m.revisions[row.ID] = row
}

func (m *documentMemStore) ListDocumentRevisions(_ context.Context, documentID uuid.UUID) ([]postgres.DocumentRevisionRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.DocumentRevisionRow
	for _, row := range m.revisions {
		if row.DocumentID == documentID {
			copy := row
			copy.Body = ""
			out = append(out, copy)
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Version > out[i].Version {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (m *documentMemStore) GetDocumentRevision(_ context.Context, revisionID uuid.UUID) (postgres.DocumentRevisionRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.revisions[revisionID]
	if !ok {
		return postgres.DocumentRevisionRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *documentMemStore) DeleteDocument(_ context.Context, documentID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.docs[documentID]; !ok {
		return postgres.ErrNotFound
	}
	for _, row := range m.docs {
		if row.ParentID != nil && *row.ParentID == documentID {
			return postgres.ErrDocumentHasChildren
		}
	}
	delete(m.docs, documentID)
	return nil
}

func TestDocumentCreateSlugFromTitleAndClash(t *testing.T) {
	ctx := context.Background()
	store := newDocumentMemStore()
	owner := store.seedUser()
	ws := store.seedWorkspace(owner.ID)
	svc := service.NewDocumentService(store)

	first, err := svc.Create(ctx, owner.ID.String(), ws.ID.String(), service.CreateDocumentInput{
		Title: "Onboarding Guide",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if first.Slug != "onboarding-guide" {
		t.Fatalf("slug: got %q", first.Slug)
	}

	second, err := svc.Create(ctx, owner.ID.String(), ws.ID.String(), service.CreateDocumentInput{
		Title: "Onboarding Guide",
	})
	if err != nil {
		t.Fatalf("create clash: %v", err)
	}
	if second.Slug != "onboarding-guide-2" {
		t.Fatalf("clash slug: got %q", second.Slug)
	}

	if _, err := svc.Create(ctx, owner.ID.String(), ws.ID.String(), service.CreateDocumentInput{
		Title: "Other",
		Slug:  "onboarding-guide",
	}); !errors.Is(err, service.ErrDocumentSlug) {
		t.Fatalf("explicit clash: got %v", err)
	}
}

func TestDocumentRejectsParentCycle(t *testing.T) {
	ctx := context.Background()
	store := newDocumentMemStore()
	owner := store.seedUser()
	ws := store.seedWorkspace(owner.ID)
	svc := service.NewDocumentService(store)

	root, err := svc.Create(ctx, owner.ID.String(), ws.ID.String(), service.CreateDocumentInput{Title: "Root"})
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	child, err := svc.Create(ctx, owner.ID.String(), ws.ID.String(), service.CreateDocumentInput{
		Title: "Child", ParentID: root.ID,
	})
	if err != nil {
		t.Fatalf("child: %v", err)
	}

	parentID := child.ID
	if _, err := svc.Update(ctx, owner.ID.String(), root.ID, service.UpdateDocumentInput{
		ParentID: &parentID,
	}); !errors.Is(err, service.ErrDocumentCycle) {
		t.Fatalf("cycle: got %v", err)
	}

	self := root.ID
	if _, err := svc.Update(ctx, owner.ID.String(), root.ID, service.UpdateDocumentInput{
		ParentID: &self,
	}); !errors.Is(err, service.ErrDocumentCycle) {
		t.Fatalf("self parent: got %v", err)
	}
}

func TestDocumentDeletePermissionAndChildren(t *testing.T) {
	ctx := context.Background()
	store := newDocumentMemStore()
	owner := store.seedUser()
	ws := store.seedWorkspace(owner.ID)
	member := postgres.UserRow{ID: id.New(), Email: "member@example.com", DisplayName: "Member"}
	store.addMember(ws.ID, member.ID, "member")
	svc := service.NewDocumentService(store)

	parent, err := svc.Create(ctx, owner.ID.String(), ws.ID.String(), service.CreateDocumentInput{Title: "Parent"})
	if err != nil {
		t.Fatalf("parent: %v", err)
	}
	if _, err := svc.Create(ctx, member.ID.String(), ws.ID.String(), service.CreateDocumentInput{
		Title: "Child", ParentID: parent.ID,
	}); err != nil {
		t.Fatalf("child: %v", err)
	}

	if err := svc.Delete(ctx, member.ID.String(), parent.ID); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("member delete: got %v", err)
	}
	if err := svc.Delete(ctx, owner.ID.String(), parent.ID); !errors.Is(err, service.ErrDocumentHasChildren) {
		t.Fatalf("owner delete with children: got %v", err)
	}

	listed, err := svc.List(ctx, owner.ID.String(), ws.ID.String())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("list count: got %d", len(listed))
	}

	var childID string
	for _, item := range listed {
		if item.ParentID == parent.ID {
			childID = item.ID
		}
	}
	if childID == "" {
		t.Fatal("missing child")
	}
	if err := svc.Delete(ctx, owner.ID.String(), childID); err != nil {
		t.Fatalf("delete child: %v", err)
	}
	if err := svc.Delete(ctx, owner.ID.String(), parent.ID); err != nil {
		t.Fatalf("delete parent: %v", err)
	}
}

func TestDocumentSearchAndGet(t *testing.T) {
	ctx := context.Background()
	store := newDocumentMemStore()
	owner := store.seedUser()
	ws := store.seedWorkspace(owner.ID)
	svc := service.NewDocumentService(store)

	created, err := svc.Create(ctx, owner.ID.String(), ws.ID.String(), service.CreateDocumentInput{
		Title: "Runbooks",
		Body:  "# Hello",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	hits, err := svc.Search(ctx, owner.ID.String(), ws.ID.String(), "run")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 || hits[0].ID != created.ID || hits[0].Body != "" {
		t.Fatalf("search hits: %+v", hits)
	}

	got, err := svc.Get(ctx, owner.ID.String(), created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Body != "# Hello" {
		t.Fatalf("body: got %q", got.Body)
	}

	stranger := postgres.UserRow{ID: id.New()}
	if _, err := svc.Get(ctx, stranger.ID.String(), created.ID); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("stranger get: got %v", err)
	}

	title := "Playbooks"
	updated, err := svc.Update(ctx, owner.ID.String(), created.ID, service.UpdateDocumentInput{Title: &title})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Title != "Playbooks" || updated.Slug != "runbooks" {
		t.Fatalf("update: %+v", updated)
	}
}

func TestDocumentRejectsNestingPastFiveLevels(t *testing.T) {
	ctx := context.Background()
	store := newDocumentMemStore()
	owner := store.seedUser()
	ws := store.seedWorkspace(owner.ID)
	svc := service.NewDocumentService(store)
	ownerID := owner.ID.String()
	wsID := ws.ID.String()

	parentID := ""
	var chain []string
	for i := 0; i < 6; i++ {
		doc, err := svc.Create(ctx, ownerID, wsID, service.CreateDocumentInput{
			Title:    "Level " + strings.Repeat("x", i+1),
			ParentID: parentID,
		})
		if err != nil {
			t.Fatalf("create depth %d: %v", i, err)
		}
		chain = append(chain, doc.ID)
		parentID = doc.ID
	}
	if _, err := svc.Create(ctx, ownerID, wsID, service.CreateDocumentInput{
		Title:    "Too deep",
		ParentID: chain[len(chain)-1],
	}); !errors.Is(err, service.ErrDocumentDepth) {
		t.Fatalf("sixth nested page: got %v", err)
	}

	otherRoot, err := svc.Create(ctx, ownerID, wsID, service.CreateDocumentInput{Title: "Other root"})
	if err != nil {
		t.Fatalf("other root: %v", err)
	}
	shallow, err := svc.Create(ctx, ownerID, wsID, service.CreateDocumentInput{
		Title: "Shallow", ParentID: otherRoot.ID,
	})
	if err != nil {
		t.Fatalf("shallow: %v", err)
	}
	moveUnderShallow := shallow.ID
	if _, err := svc.Update(ctx, ownerID, chain[1], service.UpdateDocumentInput{
		ParentID: &moveUnderShallow,
	}); !errors.Is(err, service.ErrDocumentDepth) {
		t.Fatalf("move subtree too deep: got %v", err)
	}

	moveToRoot := ""
	moved, err := svc.Update(ctx, ownerID, chain[1], service.UpdateDocumentInput{
		ParentID: &moveToRoot,
	})
	if err != nil {
		t.Fatalf("move subtree to root: %v", err)
	}
	if moved.ParentID != "" {
		t.Fatalf("expected root after move, parent %q", moved.ParentID)
	}
}

func TestDocumentRejectsInvalidInput(t *testing.T) {
	ctx := context.Background()
	store := newDocumentMemStore()
	owner := store.seedUser()
	ws := store.seedWorkspace(owner.ID)
	svc := service.NewDocumentService(store)

	if _, err := svc.Create(ctx, owner.ID.String(), ws.ID.String(), service.CreateDocumentInput{
		Title: "   ",
	}); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("empty title: got %v", err)
	}
	if _, err := svc.Create(ctx, owner.ID.String(), ws.ID.String(), service.CreateDocumentInput{
		Title: "Bad slug",
		Slug:  "Not A Slug",
	}); !errors.Is(err, service.ErrDocumentSlug) {
		t.Fatalf("bad slug: got %v", err)
	}
	if _, err := svc.Create(ctx, owner.ID.String(), ws.ID.String(), service.CreateDocumentInput{
		Title:    "Missing parent",
		ParentID: id.New().String(),
	}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("missing parent: got %v", err)
	}
	if _, err := svc.Create(ctx, owner.ID.String(), ws.ID.String(), service.CreateDocumentInput{
		Title: "Icon",
		Icon:  strings.Repeat("x", 33),
	}); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("long icon: got %v", err)
	}
}

func TestDocumentRevisionsAndRestore(t *testing.T) {
	ctx := context.Background()
	store := newDocumentMemStore()
	owner := store.seedUser()
	ws := store.seedWorkspace(owner.ID)
	svc := service.NewDocumentService(store)
	ownerID := owner.ID.String()
	wsID := ws.ID.String()

	created, err := svc.Create(ctx, ownerID, wsID, service.CreateDocumentInput{
		Title: "Guide",
		Body:  "v1",
		Icon:  "📘",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Icon != "📘" {
		t.Fatalf("icon: got %q", created.Icon)
	}

	revs, err := svc.ListRevisions(ctx, ownerID, created.ID)
	if err != nil {
		t.Fatalf("list after create: %v", err)
	}
	if len(revs) != 1 || revs[0].Version != 1 || revs[0].Title != "Guide" || revs[0].CreatedByName != "Owner" {
		t.Fatalf("create revision: %+v", revs)
	}
	if revs[0].Body != "" {
		t.Fatalf("list should omit body: %q", revs[0].Body)
	}
	if _, err := svc.RestoreRevision(ctx, ownerID, created.ID, revs[0].ID); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("sole revision restore: %v", err)
	}

	body := created.Body
	title := created.Title
	icon := created.Icon
	if _, err := svc.Update(ctx, ownerID, created.ID, service.UpdateDocumentInput{
		Title: &title, Body: &body, Icon: &icon,
	}); err != nil {
		t.Fatalf("noop update: %v", err)
	}
	revs, err = svc.ListRevisions(ctx, ownerID, created.ID)
	if err != nil {
		t.Fatalf("list after noop: %v", err)
	}
	if len(revs) != 1 {
		t.Fatalf("noop wrote a revision: %d", len(revs))
	}

	nextBody := "v2"
	updated, err := svc.Update(ctx, ownerID, created.ID, service.UpdateDocumentInput{Body: &nextBody})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Body != "v2" {
		t.Fatalf("updated body: %q", updated.Body)
	}
	revs, err = svc.ListRevisions(ctx, ownerID, created.ID)
	if err != nil {
		t.Fatalf("list after update: %v", err)
	}
	if len(revs) != 2 || revs[0].Version != 2 {
		t.Fatalf("update revisions: %+v", revs)
	}

	firstID := ""
	for _, rev := range revs {
		if rev.Version == 1 {
			firstID = rev.ID
		}
	}
	got, err := svc.GetRevision(ctx, ownerID, created.ID, firstID)
	if err != nil {
		t.Fatalf("get revision: %v", err)
	}
	if got.Body != "v1" || got.Icon != "📘" {
		t.Fatalf("get revision: %+v", got)
	}

	restored, err := svc.RestoreRevision(ctx, ownerID, created.ID, firstID)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.Body != "v1" {
		t.Fatalf("restored body: %q", restored.Body)
	}
	revs, err = svc.ListRevisions(ctx, ownerID, created.ID)
	if err != nil {
		t.Fatalf("list after restore: %v", err)
	}
	if len(revs) != 3 {
		t.Fatalf("restore revisions: %d", len(revs))
	}
}
