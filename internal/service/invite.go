package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
)

const defaultInviteTTL = 7 * 24 * time.Hour

var (
	ErrAlreadyWorkspaceMember = errors.New("already a workspace member")
	ErrInviteAlreadyPending   = errors.New("invite already pending")
	ErrInviteExpired          = errors.New("invite expired")
	ErrInviteAlreadyAccepted  = errors.New("invite already accepted")
)

type InviteStore interface {
	GetWorkspaceForUser(ctx context.Context, workspaceID, userID uuid.UUID) (postgres.WorkspaceRow, error)
	IsWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, string, error)
	GetUserByEmail(ctx context.Context, email string) (postgres.UserRow, error)
	AddWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, role domain.WorkspaceRole) error
	CreateWorkspaceInvite(ctx context.Context, workspaceID, invitedBy uuid.UUID, email, token string, role domain.WorkspaceRole, expiresAt time.Time) (postgres.WorkspaceInviteRow, error)
	ListPendingWorkspaceInvites(ctx context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceInviteRow, error)
	GetWorkspaceInviteByToken(ctx context.Context, token string) (postgres.WorkspaceInviteRow, error)
	AcceptWorkspaceInvite(ctx context.Context, inviteID uuid.UUID) error
	DeleteWorkspaceInvite(ctx context.Context, workspaceID, inviteID uuid.UUID) error
	GetUserByID(ctx context.Context, id uuid.UUID) (postgres.UserRow, error)
	ListWorkspaceMembers(ctx context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceMemberInfo, error)
	FindHomeWorkspaceForUser(ctx context.Context, userID uuid.UUID) (uuid.UUID, string, error)
	SetWorkspaceMemberOrigin(
		ctx context.Context,
		workspaceID, userID uuid.UUID,
		kind string,
		homeWorkspaceID *uuid.UUID,
		homeWorkspaceName string,
		homeServerID, homeWorkspaceRemoteID, homeWorkspaceIconURL string,
	) error
}

// InviteMailer sends (or stubs) invite emails.
type InviteMailer interface {
	EnqueueInviteEmail(ctx context.Context, to, workspaceName, acceptURL string) error
}

type noopMailer struct{}

func (noopMailer) EnqueueInviteEmail(context.Context, string, string, string) error {
	return nil
}

type InviteService struct {
	store     InviteStore
	mailer    InviteMailer
	baseURL   string
	notify    NotificationWriter
	lifecycle WorkspaceLifecycle
}

func NewInviteService(store InviteStore, baseURL string, mailer InviteMailer) *InviteService {
	if mailer == nil {
		mailer = noopMailer{}
	}
	if baseURL == "" {
		baseURL = "http://localhost:8383"
	}
	return &InviteService{
		store: store, mailer: mailer, baseURL: strings.TrimRight(baseURL, "/"),
		notify: noopNotificationWriter{},
	}
}

func (s *InviteService) WithLifecycle(lifecycle WorkspaceLifecycle) *InviteService {
	s.lifecycle = lifecycle
	return s
}

func (s *InviteService) WithNotifications(notify NotificationWriter) *InviteService {
	if notify != nil {
		s.notify = notify
	}
	return s
}

type InviteResult struct {
	Status string // "added" or "invited"
	Invite *domain.WorkspaceInvite
}

func (s *InviteService) Invite(
	ctx context.Context,
	actorID, workspaceID, email, role string,
) (*InviteResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, ErrInvalidInput
	}
	wsRole := domain.WorkspaceRoleMember
	if role != "" {
		wsRole = domain.WorkspaceRole(role)
		if wsRole != domain.WorkspaceRoleMember && wsRole != domain.WorkspaceRoleAdmin {
			return nil, ErrInvalidInput
		}
	}
	uid, err := uuid.Parse(actorID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	ws, err := s.store.GetWorkspaceForUser(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !canManageWorkspace(domain.WorkspaceRole(ws.Role)) {
		return nil, ErrForbidden
	}

	if existing, err := s.store.GetUserByEmail(ctx, email); err == nil {
		ok, _, err := s.store.IsWorkspaceMember(ctx, wid, existing.ID)
		if err != nil {
			return nil, err
		}
		if ok {
			return nil, ErrAlreadyWorkspaceMember
		}
		if err := s.store.AddWorkspaceMember(ctx, wid, existing.ID, wsRole); err != nil {
			return nil, err
		}
		_ = s.applyExternalOrigin(ctx, wid, existing.ID)
		actorName := "Someone"
		if actor, err := s.store.GetUserByID(ctx, uid); err == nil {
			actorName = actor.DisplayName
		}
		_ = s.notify.Notify(ctx, NotifyInput{
			UserID:      existing.ID.String(),
			ActorID:     uid.String(),
			Kind:        domain.NotificationWorkspaceInvite,
			WorkspaceID: wid.String(),
			Body:        fmt.Sprintf("%s added you to %s", actorName, ws.Name),
		})
		if s.lifecycle != nil {
			_ = s.lifecycle.SyncSeats(ctx, workspaceID)
		}
		return &InviteResult{Status: "added"}, nil
	} else if !errors.Is(err, postgres.ErrNotFound) {
		return nil, err
	}

	token, err := newInviteToken()
	if err != nil {
		return nil, err
	}
	row, err := s.store.CreateWorkspaceInvite(ctx, wid, uid, email, token, wsRole, time.Now().UTC().Add(defaultInviteTTL))
	if err != nil {
		if errors.Is(err, postgres.ErrInviteConflict) {
			return nil, ErrInviteAlreadyPending
		}
		return nil, err
	}
	inv := row.ToDomain()
	acceptURL := fmt.Sprintf("%s/invites/accept?token=%s", s.baseURL, token)
	_ = s.mailer.EnqueueInviteEmail(ctx, email, ws.Name, acceptURL)
	return &InviteResult{Status: "invited", Invite: &inv}, nil
}

func (s *InviteService) ListPending(ctx context.Context, actorID, workspaceID string) ([]domain.WorkspaceInvite, error) {
	uid, err := uuid.Parse(actorID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	ws, err := s.store.GetWorkspaceForUser(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !canManageWorkspace(domain.WorkspaceRole(ws.Role)) {
		return nil, ErrForbidden
	}
	rows, err := s.store.ListPendingWorkspaceInvites(ctx, wid)
	if err != nil {
		return nil, err
	}
	out := make([]domain.WorkspaceInvite, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ToDomain())
	}
	return out, nil
}

func (s *InviteService) Accept(ctx context.Context, userID, token string) (*domain.Workspace, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrInvalidInput
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	user, err := s.store.GetUserByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	inv, err := s.store.GetWorkspaceInviteByToken(ctx, token)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if inv.AcceptedAt != nil {
		return nil, ErrInviteAlreadyAccepted
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, ErrInviteExpired
	}
	if !strings.EqualFold(user.Email, inv.Email) {
		return nil, ErrForbidden
	}
	if err := s.store.AddWorkspaceMember(ctx, inv.WorkspaceID, uid, domain.WorkspaceRole(inv.Role)); err != nil {
		return nil, err
	}
	_ = s.applyExternalOrigin(ctx, inv.WorkspaceID, uid)
	if err := s.store.AcceptWorkspaceInvite(ctx, inv.ID); err != nil {
		return nil, err
	}
	ws, err := s.store.GetWorkspaceForUser(ctx, inv.WorkspaceID, uid)
	if err != nil {
		return nil, err
	}
	members, err := s.store.ListWorkspaceMembers(ctx, inv.WorkspaceID)
	if err == nil {
		for _, member := range members {
			if member.UserID == uid {
				continue
			}
			if !canManageWorkspace(domain.WorkspaceRole(member.Role)) {
				continue
			}
			_ = s.notify.Notify(ctx, NotifyInput{
				UserID:      member.UserID.String(),
				ActorID:     uid.String(),
				Kind:        domain.NotificationWorkspaceJoin,
				WorkspaceID: inv.WorkspaceID.String(),
				Body:        fmt.Sprintf("%s joined %s", user.DisplayName, ws.Name),
			})
		}
	}
	if s.lifecycle != nil {
		_ = s.lifecycle.SyncSeats(ctx, inv.WorkspaceID.String())
	}
	out := ws.ToDomain()
	return &out, nil
}

func (s *InviteService) Revoke(ctx context.Context, actorID, workspaceID, inviteID string) error {
	uid, err := uuid.Parse(actorID)
	if err != nil {
		return ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return ErrNotFound
	}
	iid, err := uuid.Parse(inviteID)
	if err != nil {
		return ErrNotFound
	}
	ws, err := s.store.GetWorkspaceForUser(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if !canManageWorkspace(domain.WorkspaceRole(ws.Role)) {
		return ErrForbidden
	}
	if err := s.store.DeleteWorkspaceInvite(ctx, wid, iid); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *InviteService) applyExternalOrigin(ctx context.Context, workspaceID, userID uuid.UUID) error {
	homeID, homeName, err := s.store.FindHomeWorkspaceForUser(ctx, userID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil
		}
		return err
	}
	if homeID == workspaceID {
		return nil
	}
	return s.store.SetWorkspaceMemberOrigin(
		ctx, workspaceID, userID, "external", &homeID, homeName, "", "", "",
	)
}

func newInviteToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
