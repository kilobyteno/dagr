package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type inviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type inviteJSON struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspaceId"`
	Email       string     `json:"email"`
	Token       string     `json:"token"`
	Role        string     `json:"role"`
	ExpiresAt   time.Time  `json:"expiresAt"`
	AcceptedAt  *time.Time `json:"acceptedAt,omitempty"`
	AcceptPath  string     `json:"acceptPath"`
}

func toInviteJSON(inv domain.WorkspaceInvite) inviteJSON {
	return inviteJSON{
		ID:          inv.ID,
		WorkspaceID: inv.WorkspaceID,
		Email:       inv.Email,
		Token:       inv.Token,
		Role:        string(inv.Role),
		ExpiresAt:   inv.ExpiresAt,
		AcceptedAt:  inv.AcceptedAt,
		AcceptPath:  "/invites/accept?token=" + inv.Token,
	}
}

func (s *Server) handleInviteToWorkspace(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	var req inviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	result, err := s.invites.Invite(r.Context(), user.ID, chi.URLParam(r, "workspaceID"), req.Email, req.Role)
	if err != nil {
		s.writeInviteError(w, r, err)
		return
	}
	resp := map[string]any{"status": result.Status}
	if result.Invite != nil {
		resp["invite"] = toInviteJSON(*result.Invite)
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleListInvites(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	items, err := s.invites.ListPending(r.Context(), user.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		s.writeInviteError(w, r, err)
		return
	}
	out := make([]inviteJSON, 0, len(items))
	for _, item := range items {
		out = append(out, toInviteJSON(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"invites": out})
}

func (s *Server) handleAcceptInvite(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	ws, err := s.invites.Accept(r.Context(), user.ID, chi.URLParam(r, "token"))
	if err != nil {
		s.writeInviteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspace": toWorkspaceJSON(*ws)})
}

func (s *Server) writeInviteError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrAlreadyWorkspaceMember):
		s.writeError(w, r, http.StatusConflict, "already_member", "That person is already in this workspace", err)
	case errors.Is(err, service.ErrInviteAlreadyPending):
		s.writeError(w, r, http.StatusConflict, "invite_pending", "An invite is already pending for that email", err)
	case errors.Is(err, service.ErrInviteExpired):
		s.writeError(w, r, http.StatusBadRequest, "invite_expired", "This invite has expired", err)
	case errors.Is(err, service.ErrInviteAlreadyAccepted):
		s.writeError(w, r, http.StatusBadRequest, "invite_accepted", "This invite has already been used", err)
	case errors.Is(err, service.ErrInvalidInput):
		s.writeError(w, r, http.StatusBadRequest, "invalid_input", "Enter a valid email address", err)
	case errors.Is(err, service.ErrNotFound):
		s.writeError(w, r, http.StatusNotFound, "not_found", "Invite not found", err)
	case errors.Is(err, service.ErrForbidden):
		s.writeError(w, r, http.StatusForbidden, "forbidden", "Invite email does not match your account", err)
	default:
		s.writeError(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong", err)
	}
}
