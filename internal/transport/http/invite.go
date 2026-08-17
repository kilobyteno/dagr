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
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	var req inviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	result, err := s.invites.Invite(r.Context(), user.ID, chi.URLParam(r, "workspaceID"), req.Email, req.Role)
	if err != nil {
		writeInviteError(w, err)
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
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	items, err := s.invites.ListPending(r.Context(), user.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		writeInviteError(w, err)
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
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	ws, err := s.invites.Accept(r.Context(), user.ID, chi.URLParam(r, "token"))
	if err != nil {
		writeInviteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspace": toWorkspaceJSON(*ws)})
}

func writeInviteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrAlreadyWorkspaceMember):
		writeError(w, http.StatusConflict, "already_member", "That person is already in this workspace")
	case errors.Is(err, service.ErrInviteAlreadyPending):
		writeError(w, http.StatusConflict, "invite_pending", "An invite is already pending for that email")
	case errors.Is(err, service.ErrInviteExpired):
		writeError(w, http.StatusBadRequest, "invite_expired", "This invite has expired")
	case errors.Is(err, service.ErrInviteAlreadyAccepted):
		writeError(w, http.StatusBadRequest, "invite_accepted", "This invite has already been used")
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", "Enter a valid email address")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Invite not found")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Invite email does not match your account")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Something went wrong")
	}
}
