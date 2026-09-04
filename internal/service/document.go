package service

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
)

var (
	ErrDocumentSlug        = errors.New("invalid document slug")
	ErrDocumentHasChildren = errors.New("document has children")
	ErrDocumentCycle       = errors.New("document parent cycle")
	ErrDocumentDepth       = errors.New("document nesting too deep")
)

const (
	maxDocumentTitleRunes = 200
	maxDocumentBodyBytes  = 256 << 10
	maxDocumentIconRunes  = 32
	documentSearchLimit   = 20
	maxDocumentDepth      = 5
)

var documentSlugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9_-]{0,78}[a-z0-9])?$`)

type DocumentStore interface {
	GetWorkspaceForUser(ctx context.Context, workspaceID, userID uuid.UUID) (postgres.WorkspaceRow, error)
	IsWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, string, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (postgres.UserRow, error)
	InsertDocument(ctx context.Context, workspaceID uuid.UUID, parentID *uuid.UUID, slug, title, body, icon string, authorID uuid.UUID) (postgres.DocumentRow, error)
	GetDocument(ctx context.Context, documentID uuid.UUID) (postgres.DocumentRow, error)
	ListDocuments(ctx context.Context, workspaceID uuid.UUID) ([]postgres.DocumentRow, error)
	SearchDocuments(ctx context.Context, workspaceID uuid.UUID, query string, limit int) ([]postgres.DocumentRow, error)
	DocumentSlugExists(ctx context.Context, workspaceID uuid.UUID, slug string, exceptID *uuid.UUID) (bool, error)
	CountDocumentChildren(ctx context.Context, documentID uuid.UUID) (int, error)
	UpdateDocument(ctx context.Context, documentID uuid.UUID, parentID *uuid.UUID, slug, title, body, icon string, updatedBy uuid.UUID) (postgres.DocumentRow, error)
	DeleteDocument(ctx context.Context, documentID uuid.UUID) error
	ListDocumentRevisions(ctx context.Context, documentID uuid.UUID) ([]postgres.DocumentRevisionRow, error)
	GetDocumentRevision(ctx context.Context, revisionID uuid.UUID) (postgres.DocumentRevisionRow, error)
}

type DocumentService struct {
	store DocumentStore
}

func NewDocumentService(store DocumentStore) *DocumentService {
	return &DocumentService{store: store}
}

type CreateDocumentInput struct {
	Title    string
	Body     string
	Slug     string
	Icon     string
	ParentID string
}

type UpdateDocumentInput struct {
	Title    *string
	Body     *string
	Slug     *string
	Icon     *string
	ParentID *string
}

func (s *DocumentService) List(ctx context.Context, userID, workspaceID string) ([]domain.Document, error) {
	_, wid, err := s.requireMember(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.ListDocuments(ctx, wid)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Document, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ToDomain())
	}
	return out, nil
}

func (s *DocumentService) Search(ctx context.Context, userID, workspaceID, query string) ([]domain.Document, error) {
	_, wid, err := s.requireMember(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return s.List(ctx, userID, workspaceID)
	}
	if utf8.RuneCountInString(query) > 80 {
		return nil, ErrInvalidInput
	}
	rows, err := s.store.SearchDocuments(ctx, wid, query, documentSearchLimit)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Document, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ToDomain())
	}
	return out, nil
}

func (s *DocumentService) Get(ctx context.Context, userID, documentID string) (*domain.Document, error) {
	uid, err := parseUserID(userID)
	if err != nil {
		return nil, err
	}
	did, err := uuid.Parse(documentID)
	if err != nil {
		return nil, ErrNotFound
	}
	row, err := s.store.GetDocument(ctx, did)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := s.requireMemberOfWorkspace(ctx, uid, row.WorkspaceID); err != nil {
		return nil, err
	}
	out := row.ToDomain()
	return &out, nil
}

func (s *DocumentService) Create(ctx context.Context, userID, workspaceID string, input CreateDocumentInput) (*domain.Document, error) {
	uid, wid, err := s.requireMember(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	title, err := normaliseDocumentTitle(input.Title)
	if err != nil {
		return nil, err
	}
	body, err := normaliseDocumentBody(input.Body)
	if err != nil {
		return nil, err
	}
	icon, err := normaliseDocumentIcon(input.Icon)
	if err != nil {
		return nil, err
	}
	parentID, err := s.resolveParent(ctx, wid, "", input.ParentID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureDepth(ctx, wid, "", parentID); err != nil {
		return nil, err
	}
	slug, err := s.allocateSlug(ctx, wid, input.Slug, title, nil)
	if err != nil {
		return nil, err
	}
	row, err := s.store.InsertDocument(ctx, wid, parentID, slug, title, body, icon, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrDocumentSlugConflict) {
			return nil, ErrDocumentSlug
		}
		return nil, err
	}
	out := row.ToDomain()
	return &out, nil
}

func (s *DocumentService) Update(ctx context.Context, userID, documentID string, input UpdateDocumentInput) (*domain.Document, error) {
	uid, err := parseUserID(userID)
	if err != nil {
		return nil, err
	}
	did, err := uuid.Parse(documentID)
	if err != nil {
		return nil, ErrNotFound
	}
	row, err := s.store.GetDocument(ctx, did)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := s.requireMemberOfWorkspace(ctx, uid, row.WorkspaceID); err != nil {
		return nil, err
	}

	title := row.Title
	if input.Title != nil {
		title, err = normaliseDocumentTitle(*input.Title)
		if err != nil {
			return nil, err
		}
	}
	body := row.Body
	if input.Body != nil {
		body, err = normaliseDocumentBody(*input.Body)
		if err != nil {
			return nil, err
		}
	}
	icon := row.Icon
	if input.Icon != nil {
		icon, err = normaliseDocumentIcon(*input.Icon)
		if err != nil {
			return nil, err
		}
	}
	parentID := row.ParentID
	if input.ParentID != nil {
		parentID, err = s.resolveParent(ctx, row.WorkspaceID, row.ID.String(), *input.ParentID)
		if err != nil {
			return nil, err
		}
		if err := s.ensureDepth(ctx, row.WorkspaceID, row.ID.String(), parentID); err != nil {
			return nil, err
		}
	}
	slug := row.Slug
	if input.Slug != nil {
		slug, err = s.allocateSlug(ctx, row.WorkspaceID, *input.Slug, title, &did)
		if err != nil {
			return nil, err
		}
	}
	if title == row.Title && body == row.Body && icon == row.Icon && slug == row.Slug && sameParentID(parentID, row.ParentID) {
		out := row.ToDomain()
		return &out, nil
	}

	updated, err := s.store.UpdateDocument(ctx, did, parentID, slug, title, body, icon, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrDocumentSlugConflict) {
			return nil, ErrDocumentSlug
		}
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	out := updated.ToDomain()
	return &out, nil
}

func (s *DocumentService) Delete(ctx context.Context, userID, documentID string) error {
	uid, err := parseUserID(userID)
	if err != nil {
		return err
	}
	did, err := uuid.Parse(documentID)
	if err != nil {
		return ErrNotFound
	}
	row, err := s.store.GetDocument(ctx, did)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	ws, err := s.store.GetWorkspaceForUser(ctx, row.WorkspaceID, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	role := domain.WorkspaceRole(ws.Role)
	if role != domain.WorkspaceRoleOwner && role != domain.WorkspaceRoleAdmin {
		return ErrForbidden
	}
	children, err := s.store.CountDocumentChildren(ctx, did)
	if err != nil {
		return err
	}
	if children > 0 {
		return ErrDocumentHasChildren
	}
	if err := s.store.DeleteDocument(ctx, did); err != nil {
		if errors.Is(err, postgres.ErrDocumentHasChildren) {
			return ErrDocumentHasChildren
		}
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *DocumentService) ListRevisions(ctx context.Context, userID, documentID string) ([]domain.DocumentRevision, error) {
	_, row, err := s.requireDocumentMember(ctx, userID, documentID)
	if err != nil {
		return nil, err
	}
	revs, err := s.store.ListDocumentRevisions(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return s.revisionsWithNames(ctx, revs), nil
}

func (s *DocumentService) GetRevision(ctx context.Context, userID, documentID, revisionID string) (*domain.DocumentRevision, error) {
	_, row, err := s.requireDocumentMember(ctx, userID, documentID)
	if err != nil {
		return nil, err
	}
	rev, err := s.loadRevision(ctx, row.ID, revisionID)
	if err != nil {
		return nil, err
	}
	out := s.revisionWithName(ctx, rev)
	return &out, nil
}

func (s *DocumentService) RestoreRevision(ctx context.Context, userID, documentID, revisionID string) (*domain.Document, error) {
	_, row, err := s.requireDocumentMember(ctx, userID, documentID)
	if err != nil {
		return nil, err
	}
	revs, err := s.store.ListDocumentRevisions(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	if len(revs) < 2 {
		return nil, ErrInvalidInput
	}
	rev, err := s.loadRevision(ctx, row.ID, revisionID)
	if err != nil {
		return nil, err
	}
	parentID := ""
	if rev.ParentID != nil {
		parentID = rev.ParentID.String()
	}
	return s.Update(ctx, userID, documentID, UpdateDocumentInput{
		Title:    &rev.Title,
		Body:     &rev.Body,
		Slug:     &rev.Slug,
		Icon:     &rev.Icon,
		ParentID: &parentID,
	})
}

func (s *DocumentService) requireDocumentMember(ctx context.Context, userID, documentID string) (uuid.UUID, postgres.DocumentRow, error) {
	uid, err := parseUserID(userID)
	if err != nil {
		return uuid.Nil, postgres.DocumentRow{}, err
	}
	did, err := uuid.Parse(documentID)
	if err != nil {
		return uuid.Nil, postgres.DocumentRow{}, ErrNotFound
	}
	row, err := s.store.GetDocument(ctx, did)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return uuid.Nil, postgres.DocumentRow{}, ErrNotFound
		}
		return uuid.Nil, postgres.DocumentRow{}, err
	}
	if err := s.requireMemberOfWorkspace(ctx, uid, row.WorkspaceID); err != nil {
		return uuid.Nil, postgres.DocumentRow{}, err
	}
	return uid, row, nil
}

func (s *DocumentService) loadRevision(ctx context.Context, documentID uuid.UUID, revisionID string) (postgres.DocumentRevisionRow, error) {
	rid, err := uuid.Parse(revisionID)
	if err != nil {
		return postgres.DocumentRevisionRow{}, ErrNotFound
	}
	rev, err := s.store.GetDocumentRevision(ctx, rid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return postgres.DocumentRevisionRow{}, ErrNotFound
		}
		return postgres.DocumentRevisionRow{}, err
	}
	if rev.DocumentID != documentID {
		return postgres.DocumentRevisionRow{}, ErrNotFound
	}
	return rev, nil
}

func (s *DocumentService) revisionsWithNames(ctx context.Context, rows []postgres.DocumentRevisionRow) []domain.DocumentRevision {
	out := make([]domain.DocumentRevision, 0, len(rows))
	names := map[uuid.UUID]string{}
	for _, row := range rows {
		rev := row.ToDomain()
		if name, ok := names[row.CreatedBy]; ok {
			rev.CreatedByName = name
		} else if user, err := s.store.GetUserByID(ctx, row.CreatedBy); err == nil {
			names[row.CreatedBy] = user.DisplayName
			rev.CreatedByName = user.DisplayName
		}
		out = append(out, rev)
	}
	return out
}

func (s *DocumentService) revisionWithName(ctx context.Context, row postgres.DocumentRevisionRow) domain.DocumentRevision {
	out := row.ToDomain()
	if user, err := s.store.GetUserByID(ctx, row.CreatedBy); err == nil {
		out.CreatedByName = user.DisplayName
	}
	return out
}

func (s *DocumentService) requireMember(ctx context.Context, userID, workspaceID string) (uuid.UUID, uuid.UUID, error) {
	uid, err := parseUserID(userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrNotFound
	}
	if err := s.requireMemberOfWorkspace(ctx, uid, wid); err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return uid, wid, nil
}

func (s *DocumentService) requireMemberOfWorkspace(ctx context.Context, userID, workspaceID uuid.UUID) error {
	ok, _, err := s.store.IsWorkspaceMember(ctx, workspaceID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func parseUserID(userID string) (uuid.UUID, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, ErrInvalidInput
	}
	return uid, nil
}

func (s *DocumentService) resolveParent(ctx context.Context, workspaceID uuid.UUID, documentID, parentID string) (*uuid.UUID, error) {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return nil, nil
	}
	pid, err := uuid.Parse(parentID)
	if err != nil {
		return nil, ErrNotFound
	}
	if documentID != "" && pid.String() == documentID {
		return nil, ErrDocumentCycle
	}
	parent, err := s.store.GetDocument(ctx, pid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if parent.WorkspaceID != workspaceID {
		return nil, ErrNotFound
	}
	if documentID != "" {
		if err := s.ensureNoCycle(ctx, documentID, &pid); err != nil {
			return nil, err
		}
	}
	return &pid, nil
}

func (s *DocumentService) ensureDepth(ctx context.Context, workspaceID uuid.UUID, documentID string, parentID *uuid.UUID) error {
	depth, err := s.parentChainLength(ctx, parentID)
	if err != nil {
		return err
	}
	extra := 0
	if documentID != "" {
		did, parseErr := uuid.Parse(documentID)
		if parseErr != nil {
			return ErrNotFound
		}
		extra, err = s.subtreeHeight(ctx, workspaceID, did)
		if err != nil {
			return err
		}
	}
	if depth+extra > maxDocumentDepth {
		return ErrDocumentDepth
	}
	return nil
}

func (s *DocumentService) parentChainLength(ctx context.Context, parentID *uuid.UUID) (int, error) {
	depth := 0
	current := parentID
	seen := map[string]struct{}{}
	for current != nil {
		key := current.String()
		if _, ok := seen[key]; ok {
			return 0, ErrDocumentCycle
		}
		seen[key] = struct{}{}
		row, err := s.store.GetDocument(ctx, *current)
		if err != nil {
			if errors.Is(err, postgres.ErrNotFound) {
				return 0, ErrNotFound
			}
			return 0, err
		}
		depth++
		if depth > maxDocumentDepth {
			return depth, nil
		}
		current = row.ParentID
	}
	return depth, nil
}

func (s *DocumentService) subtreeHeight(ctx context.Context, workspaceID, documentID uuid.UUID) (int, error) {
	rows, err := s.store.ListDocuments(ctx, workspaceID)
	if err != nil {
		return 0, err
	}
	children := map[string][]uuid.UUID{}
	for _, row := range rows {
		if row.ParentID == nil {
			continue
		}
		key := row.ParentID.String()
		children[key] = append(children[key], row.ID)
	}
	var height func(id uuid.UUID) int
	height = func(id uuid.UUID) int {
		kids := children[id.String()]
		if len(kids) == 0 {
			return 0
		}
		maxH := 0
		for _, kid := range kids {
			if h := height(kid); h > maxH {
				maxH = h
			}
		}
		return 1 + maxH
	}
	return height(documentID), nil
}

func (s *DocumentService) ensureNoCycle(ctx context.Context, documentID string, parentID *uuid.UUID) error {
	seen := map[string]struct{}{documentID: {}}
	current := parentID
	for current != nil {
		key := current.String()
		if _, ok := seen[key]; ok {
			return ErrDocumentCycle
		}
		seen[key] = struct{}{}
		row, err := s.store.GetDocument(ctx, *current)
		if err != nil {
			if errors.Is(err, postgres.ErrNotFound) {
				return ErrNotFound
			}
			return err
		}
		current = row.ParentID
	}
	return nil
}

func (s *DocumentService) allocateSlug(ctx context.Context, workspaceID uuid.UUID, raw, title string, exceptID *uuid.UUID) (string, error) {
	if strings.TrimSpace(raw) != "" {
		slug, err := normaliseDocumentSlug(raw)
		if err != nil {
			return "", err
		}
		exists, err := s.store.DocumentSlugExists(ctx, workspaceID, slug, exceptID)
		if err != nil {
			return "", err
		}
		if exists {
			return "", ErrDocumentSlug
		}
		return slug, nil
	}

	base := slugifyDocumentTitle(title)
	candidate := base
	for i := 2; i < 100; i++ {
		exists, err := s.store.DocumentSlugExists(ctx, workspaceID, candidate, exceptID)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		next := truncateSlug(base, i)
		if !documentSlugPattern.MatchString(next) {
			return "", ErrDocumentSlug
		}
		candidate = next
	}
	return "", ErrDocumentSlug
}

func truncateSlug(base string, n int) string {
	suffix := "-" + strconv.Itoa(n)
	maxBase := 80 - len(suffix)
	if maxBase < 1 {
		return base + suffix
	}
	trimmed := base
	if len(trimmed) > maxBase {
		trimmed = strings.TrimRight(trimmed[:maxBase], "-_")
	}
	if trimmed == "" {
		trimmed = "page"
	}
	return trimmed + suffix
}

func normaliseDocumentTitle(raw string) (string, error) {
	title := strings.TrimSpace(raw)
	if title == "" {
		return "", ErrInvalidInput
	}
	if utf8.RuneCountInString(title) > maxDocumentTitleRunes {
		return "", ErrInvalidInput
	}
	return title, nil
}

func normaliseDocumentBody(raw string) (string, error) {
	if len(raw) > maxDocumentBodyBytes {
		return "", ErrInvalidInput
	}
	return raw, nil
}

func normaliseDocumentIcon(raw string) (string, error) {
	icon := strings.TrimSpace(raw)
	if utf8.RuneCountInString(icon) > maxDocumentIconRunes {
		return "", ErrInvalidInput
	}
	return icon, nil
}

func sameParentID(a, b *uuid.UUID) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func normaliseDocumentSlug(raw string) (string, error) {
	slug := strings.TrimSpace(strings.ToLower(raw))
	if slug == "" || !documentSlugPattern.MatchString(slug) {
		return "", ErrDocumentSlug
	}
	return slug, nil
}

func slugifyDocumentTitle(title string) string {
	lower := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return '-'
		}
		if r >= 'A' && r <= 'Z' {
			return unicode.ToLower(r)
		}
		return r
	}, title)
	var b strings.Builder
	lastHyphen := false
	for _, r := range lower {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
			continue
		}
		if r == '-' || r == '_' {
			if lastHyphen || b.Len() == 0 {
				continue
			}
			b.WriteRune(r)
			lastHyphen = true
			continue
		}
		b.WriteRune(r)
		lastHyphen = false
	}
	slug := strings.Trim(b.String(), "-_")
	if len(slug) > 80 {
		slug = strings.Trim(slug[:80], "-_")
	}
	if slug == "" {
		return "page"
	}
	if !documentSlugPattern.MatchString(slug) {
		return "page"
	}
	return slug
}
