package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
)

var (
	ErrDomainDenied      = errors.New("domain is not allowed")
	ErrDomainUnverified  = errors.New("domain is not verified")
	ErrDomainDNSMismatch = errors.New("dns verification record not found")
	ErrDomainConflict    = errors.New("domain already claimed")
)

const domainVerificationPrefix = "dagr-domain-verification="

var domainNamePattern = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)

var freeMailDomains = map[string]struct{}{
	"gmail.com":      {},
	"googlemail.com": {},
	"outlook.com":    {},
	"hotmail.com":    {},
	"live.com":       {},
	"msn.com":        {},
	"yahoo.com":      {},
	"ymail.com":      {},
	"icloud.com":     {},
	"me.com":         {},
	"mac.com":        {},
	"proton.me":      {},
	"protonmail.com": {},
	"aol.com":        {},
	"mail.com":       {},
	"gmx.com":        {},
	"gmx.net":        {},
	"zoho.com":       {},
	"yandex.com":     {},
	"yandex.ru":      {},
}

// TXTResolver looks up DNS TXT records (injectable for tests).
type TXTResolver func(host string) ([]string, error)

type DomainStore interface {
	GetWorkspaceForUser(ctx context.Context, workspaceID, userID uuid.UUID) (postgres.WorkspaceRow, error)
	CreateWorkspaceDomain(ctx context.Context, workspaceID, createdBy uuid.UUID, domainName, verificationToken string) (postgres.WorkspaceDomainRow, error)
	ListWorkspaceDomains(ctx context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceDomainRow, error)
	GetWorkspaceDomain(ctx context.Context, workspaceID, domainID uuid.UUID) (postgres.WorkspaceDomainRow, error)
	MarkWorkspaceDomainVerified(ctx context.Context, workspaceID, domainID uuid.UUID) (postgres.WorkspaceDomainRow, error)
	UpdateWorkspaceDomainAutoJoin(ctx context.Context, workspaceID, domainID uuid.UUID, autoJoin bool) (postgres.WorkspaceDomainRow, error)
	DeleteWorkspaceDomain(ctx context.Context, workspaceID, domainID uuid.UUID) error
	DomainVerifiedElsewhere(ctx context.Context, domainName string, excludeWorkspaceID uuid.UUID) (bool, error)
	ListAutoJoinWorkspaceIDsByDomain(ctx context.Context, domainName string) ([]uuid.UUID, error)
	AddWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, role domain.WorkspaceRole) error
}

type DomainService struct {
	store     DomainStore
	lookupTXT TXTResolver
	lifecycle WorkspaceLifecycle
}

func NewDomainService(store DomainStore) *DomainService {
	return &DomainService{
		store: store,
		lookupTXT: func(host string) ([]string, error) {
			return net.LookupTXT(host)
		},
	}
}

func (s *DomainService) WithLifecycle(lifecycle WorkspaceLifecycle) *DomainService {
	s.lifecycle = lifecycle
	return s
}

// WithTXTResolver replaces the DNS lookup used for verification (tests).
func (s *DomainService) WithTXTResolver(fn TXTResolver) *DomainService {
	s.lookupTXT = fn
	return s
}

type DomainDNSInstructions struct {
	Host  string `json:"host"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

func DomainTXTHost(domainName string) string {
	return "_dagr-challenge." + domainName
}

func DomainTXTValue(token string) string {
	return domainVerificationPrefix + token
}

func (s *DomainService) List(ctx context.Context, userID, workspaceID string) ([]domain.WorkspaceDomain, error) {
	if _, err := s.requireMember(ctx, userID, workspaceID); err != nil {
		return nil, err
	}
	wid, _ := uuid.Parse(workspaceID)
	rows, err := s.store.ListWorkspaceDomains(ctx, wid)
	if err != nil {
		return nil, err
	}
	out := make([]domain.WorkspaceDomain, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ToDomain())
	}
	return out, nil
}

func (s *DomainService) Add(ctx context.Context, userID, workspaceID, domainName string) (*domain.WorkspaceDomain, error) {
	ws, err := s.requireManager(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	normalised, err := normaliseDomain(domainName)
	if err != nil {
		return nil, err
	}
	token, err := newVerificationToken()
	if err != nil {
		return nil, err
	}
	uid, _ := uuid.Parse(userID)
	row, err := s.store.CreateWorkspaceDomain(ctx, ws.ID, uid, normalised, token)
	if err != nil {
		if errors.Is(err, postgres.ErrDomainConflict) {
			return nil, ErrDomainConflict
		}
		return nil, err
	}
	d := row.ToDomain()
	return &d, nil
}

func (s *DomainService) Verify(ctx context.Context, userID, workspaceID, domainID string) (*domain.WorkspaceDomain, error) {
	ws, err := s.requireManager(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	did, err := uuid.Parse(domainID)
	if err != nil {
		return nil, ErrNotFound
	}
	row, err := s.store.GetWorkspaceDomain(ctx, ws.ID, did)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if row.VerifiedAt != nil {
		d := row.ToDomain()
		return &d, nil
	}

	taken, err := s.store.DomainVerifiedElsewhere(ctx, row.Domain, ws.ID)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrDomainConflict
	}

	host := DomainTXTHost(row.Domain)
	records, err := s.lookupTXT(host)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDomainDNSMismatch, err)
	}
	want := DomainTXTValue(row.VerificationToken)
	if !txtContains(records, want) {
		return nil, ErrDomainDNSMismatch
	}

	updated, err := s.store.MarkWorkspaceDomainVerified(ctx, ws.ID, did)
	if err != nil {
		if errors.Is(err, postgres.ErrDomainVerifiedConflict) {
			return nil, ErrDomainConflict
		}
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	d := updated.ToDomain()
	return &d, nil
}

func (s *DomainService) SetAutoJoin(
	ctx context.Context,
	userID, workspaceID, domainID string,
	autoJoin bool,
) (*domain.WorkspaceDomain, error) {
	ws, err := s.requireManager(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	did, err := uuid.Parse(domainID)
	if err != nil {
		return nil, ErrNotFound
	}
	current, err := s.store.GetWorkspaceDomain(ctx, ws.ID, did)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if current.VerifiedAt == nil {
		return nil, ErrDomainUnverified
	}
	updated, err := s.store.UpdateWorkspaceDomainAutoJoin(ctx, ws.ID, did, autoJoin)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	d := updated.ToDomain()
	return &d, nil
}

func (s *DomainService) Delete(ctx context.Context, userID, workspaceID, domainID string) error {
	ws, err := s.requireManager(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	did, err := uuid.Parse(domainID)
	if err != nil {
		return ErrNotFound
	}
	if err := s.store.DeleteWorkspaceDomain(ctx, ws.ID, did); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// AutoJoinForEmail adds the user to workspaces with a verified auto-join domain matching the email.
func (s *DomainService) AutoJoinForEmail(ctx context.Context, userID uuid.UUID, email string) error {
	domainName, err := emailDomain(email)
	if err != nil {
		return nil
	}
	workspaceIDs, err := s.store.ListAutoJoinWorkspaceIDsByDomain(ctx, domainName)
	if err != nil {
		return err
	}
	for _, workspaceID := range workspaceIDs {
		if err := s.store.AddWorkspaceMember(ctx, workspaceID, userID, domain.WorkspaceRoleMember); err != nil {
			return err
		}
		if s.lifecycle != nil {
			_ = s.lifecycle.SyncSeats(ctx, workspaceID.String())
		}
	}
	return nil
}

func (s *DomainService) requireMember(ctx context.Context, userID, workspaceID string) (postgres.WorkspaceRow, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return postgres.WorkspaceRow{}, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return postgres.WorkspaceRow{}, ErrNotFound
	}
	ws, err := s.store.GetWorkspaceForUser(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return postgres.WorkspaceRow{}, ErrNotFound
		}
		return postgres.WorkspaceRow{}, err
	}
	return ws, nil
}

func (s *DomainService) requireManager(ctx context.Context, userID, workspaceID string) (postgres.WorkspaceRow, error) {
	ws, err := s.requireMember(ctx, userID, workspaceID)
	if err != nil {
		return postgres.WorkspaceRow{}, err
	}
	if !canManageWorkspace(domain.WorkspaceRole(ws.Role)) {
		return postgres.WorkspaceRow{}, ErrForbidden
	}
	return ws, nil
}

func normaliseDomain(raw string) (string, error) {
	d := strings.TrimSpace(strings.ToLower(raw))
	d = strings.TrimPrefix(d, "@")
	d = strings.Trim(d, ".")
	if d == "" || !domainNamePattern.MatchString(d) {
		return "", ErrInvalidInput
	}
	if _, blocked := freeMailDomains[d]; blocked {
		return "", ErrDomainDenied
	}
	return d, nil
}

func emailDomain(email string) (string, error) {
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return "", ErrInvalidInput
	}
	return normaliseDomain(email[at+1:])
}

func newVerificationToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func txtContains(records []string, want string) bool {
	for _, rec := range records {
		// DNS TXT may be split; join is already done by LookupTXT per string.
		if strings.Contains(strings.TrimSpace(rec), want) {
			return true
		}
	}
	return false
}
