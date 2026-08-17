package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type updateMemberRoleRequest struct {
	Role string `json:"role"`
}

type transferOwnershipRequest struct {
	UserID string `json:"userId"`
}

func (s *Server) handleUpdateWorkspaceMemberRole(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	var req updateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	member, err := s.workspaces.UpdateMemberRole(
		r.Context(),
		user.ID,
		chi.URLParam(r, "workspaceID"),
		chi.URLParam(r, "userID"),
		req.Role,
	)
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	enriched := s.withPresence(r.Context(), *member)
	writeJSON(w, http.StatusOK, workspaceMemberResponse{Member: toWorkspaceMemberJSON(enriched)})
}

func (s *Server) handleRemoveWorkspaceMember(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	if err := s.workspaces.RemoveMember(
		r.Context(),
		user.ID,
		chi.URLParam(r, "workspaceID"),
		chi.URLParam(r, "userID"),
	); err != nil {
		writeWorkspaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLeaveWorkspace(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	if err := s.workspaces.Leave(r.Context(), user.ID, chi.URLParam(r, "workspaceID")); err != nil {
		writeWorkspaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTransferWorkspaceOwnership(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	var req transferOwnershipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	if err := s.workspaces.TransferOwnership(
		r.Context(), user.ID, chi.URLParam(r, "workspaceID"), req.UserID,
	); err != nil {
		writeWorkspaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRevokeInvite(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	if err := s.invites.Revoke(
		r.Context(),
		user.ID,
		chi.URLParam(r, "workspaceID"),
		chi.URLParam(r, "inviteID"),
	); err != nil {
		writeInviteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
