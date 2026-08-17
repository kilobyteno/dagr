package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type createWorkspaceRequest struct {
	Name string `json:"name"`
}

type renameWorkspaceRequest struct {
	Name string `json:"name"`
}

type workspaceJSON struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Slug          string     `json:"slug"`
	Role          string     `json:"role"`
	HasIcon       bool       `json:"hasIcon"`
	IconUpdatedAt *time.Time `json:"iconUpdatedAt,omitempty"`
}

type createWorkspaceResponse struct {
	Workspace workspaceJSON `json:"workspace"`
	Channels  []channelJSON `json:"channels"`
}

type listWorkspacesResponse struct {
	Workspaces []workspaceJSON `json:"workspaces"`
}

type getWorkspaceResponse struct {
	Workspace workspaceJSON `json:"workspace"`
}

type listChannelsResponse struct {
	Channels []channelJSON `json:"channels"`
}

type workspaceMemberJSON struct {
	UserID                string     `json:"userId"`
	DisplayName           string     `json:"displayName"`
	Handle                string     `json:"handle"`
	FormerHandles         []string   `json:"formerHandles,omitempty"`
	Role                  string     `json:"role"`
	Kind                  string     `json:"kind,omitempty"`
	IsExternal            bool       `json:"isExternal,omitempty"`
	HomeWorkspaceID       string     `json:"homeWorkspaceId,omitempty"`
	HomeWorkspaceName     string     `json:"homeWorkspaceName,omitempty"`
	HomeServerID          string     `json:"homeServerId,omitempty"`
	HomeWorkspaceRemoteID string     `json:"homeWorkspaceRemoteId,omitempty"`
	HomeWorkspaceIconURL  string     `json:"homeWorkspaceIconUrl,omitempty"`
	HomeServerURL         string     `json:"homeServerUrl,omitempty"`
	StatusEmoji           string     `json:"statusEmoji,omitempty"`
	StatusText            string     `json:"statusText,omitempty"`
	StatusExpiresAt       *time.Time `json:"statusExpiresAt,omitempty"`
	Presence              string     `json:"presence,omitempty"`
	HasAvatar             bool       `json:"hasAvatar"`
	AvatarUpdatedAt       *time.Time `json:"avatarUpdatedAt,omitempty"`
}

type workspaceMemberResponse struct {
	Member workspaceMemberJSON `json:"member"`
}

type listWorkspaceMembersResponse struct {
	Members []workspaceMemberJSON `json:"members"`
}

type updateWorkspaceMeRequest struct {
	Handle string `json:"handle"`
}

func toWorkspaceMemberJSON(m domain.WorkspaceMember) workspaceMemberJSON {
	presenceState := string(m.Presence)
	if presenceState == "" {
		presenceState = string(domain.PresenceOffline)
	}
	return workspaceMemberJSON{
		UserID:                m.UserID,
		DisplayName:           m.DisplayName,
		Handle:                m.Handle,
		FormerHandles:         m.FormerHandles,
		Role:                  string(m.Role),
		Kind:                  m.Kind,
		IsExternal:            m.IsExternal,
		HomeWorkspaceID:       m.HomeWorkspaceID,
		HomeWorkspaceName:     m.HomeWorkspaceName,
		HomeServerID:          m.HomeServerID,
		HomeWorkspaceRemoteID: m.HomeWorkspaceRemoteID,
		HomeWorkspaceIconURL:  m.HomeWorkspaceIconURL,
		HomeServerURL:         m.HomeServerURL,
		StatusEmoji:           m.StatusEmoji,
		StatusText:            m.StatusText,
		StatusExpiresAt:       m.StatusExpiresAt,
		Presence:              presenceState,
		HasAvatar:             m.HasAvatar,
		AvatarUpdatedAt:       m.AvatarUpdatedAt,
	}
}

func (s *Server) withPresence(ctx context.Context, member domain.WorkspaceMember) domain.WorkspaceMember {
	if s.presence == nil {
		member.Presence = domain.PresenceOffline
		return member
	}
	member.Presence = domain.PresenceState(s.presence.Get(ctx, member.UserID))
	return member
}

func toWorkspaceJSON(w domain.Workspace) workspaceJSON {
	return workspaceJSON{
		ID:            w.ID,
		Name:          w.Name,
		Slug:          w.Slug,
		Role:          string(w.Role),
		HasIcon:       w.HasIcon,
		IconUpdatedAt: w.IconUpdatedAt,
	}
}

func toChannelJSON(c domain.Channel) channelJSON {
	return toChannelJSONFull(c)
}

func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	items, err := s.workspaces.ListForUser(r.Context(), user.ID)
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	out := make([]workspaceJSON, 0, len(items))
	for _, item := range items {
		out = append(out, toWorkspaceJSON(item))
	}
	writeJSON(w, http.StatusOK, listWorkspacesResponse{Workspaces: out})
}

func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	var req createWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	result, err := s.workspaces.Create(r.Context(), user.ID, req.Name)
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	channels := make([]channelJSON, 0, len(result.Channels))
	for _, ch := range result.Channels {
		channels = append(channels, toChannelJSON(ch))
	}
	writeJSON(w, http.StatusCreated, createWorkspaceResponse{
		Workspace: toWorkspaceJSON(result.Workspace),
		Channels:  channels,
	})
}

func (s *Server) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	ws, err := s.workspaces.Get(r.Context(), user.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, getWorkspaceResponse{Workspace: toWorkspaceJSON(*ws)})
}

func (s *Server) handleRenameWorkspace(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	var req renameWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	ws, err := s.workspaces.Rename(r.Context(), user.ID, chi.URLParam(r, "workspaceID"), req.Name)
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, getWorkspaceResponse{Workspace: toWorkspaceJSON(*ws)})
}

func (s *Server) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	if err := s.workspaces.Delete(r.Context(), user.ID, chi.URLParam(r, "workspaceID")); err != nil {
		writeWorkspaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePutWorkspaceIcon(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	const maxMemory = 2 << 20
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_icon", "Expected a multipart image upload")
		return
	}
	file, header, err := r.FormFile("icon")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_icon", "Missing icon file")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	ws, err := s.workspaces.SetIcon(
		r.Context(),
		user.ID,
		chi.URLParam(r, "workspaceID"),
		file,
		contentType,
	)
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, getWorkspaceResponse{Workspace: toWorkspaceJSON(*ws)})
}

func (s *Server) handleDeleteWorkspaceIcon(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	ws, err := s.workspaces.ClearIcon(r.Context(), user.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, getWorkspaceResponse{Workspace: toWorkspaceJSON(*ws)})
}

func (s *Server) handleGetWorkspaceIcon(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	icon, err := s.workspaces.GetIcon(r.Context(), user.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	w.Header().Set("Content-Type", icon.ContentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	if !icon.UpdatedAt.IsZero() {
		w.Header().Set("Last-Modified", icon.UpdatedAt.UTC().Format(http.TimeFormat))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, bytes.NewReader(icon.Bytes))
}

func (s *Server) handleListChannels(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	items, err := s.workspaces.ListChannels(r.Context(), user.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	out := make([]channelJSON, 0, len(items))
	for _, item := range items {
		out = append(out, toChannelJSON(item))
	}
	writeJSON(w, http.StatusOK, listChannelsResponse{Channels: out})
}

func (s *Server) handleGetWorkspaceMe(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	member, err := s.workspaces.GetMyMembership(r.Context(), user.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	enriched := s.withPresence(r.Context(), *member)
	writeJSON(w, http.StatusOK, workspaceMemberResponse{Member: toWorkspaceMemberJSON(enriched)})
}

func (s *Server) handleUpdateWorkspaceMe(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	var req updateWorkspaceMeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	member, err := s.workspaces.UpdateMyHandle(
		r.Context(),
		user.ID,
		chi.URLParam(r, "workspaceID"),
		req.Handle,
	)
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	enriched := s.withPresence(r.Context(), *member)
	writeJSON(w, http.StatusOK, workspaceMemberResponse{Member: toWorkspaceMemberJSON(enriched)})
}

func (s *Server) handleListWorkspaceMembers(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	items, err := s.workspaces.ListMembers(r.Context(), user.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.UserID)
	}
	presenceByUser := map[string]string{}
	if s.presence != nil {
		for id, state := range s.presence.GetMany(r.Context(), ids) {
			presenceByUser[id] = string(state)
		}
	}
	out := make([]workspaceMemberJSON, 0, len(items))
	for _, item := range items {
		if presence, ok := presenceByUser[item.UserID]; ok {
			item.Presence = domain.PresenceState(presence)
		} else {
			item.Presence = domain.PresenceOffline
		}
		out = append(out, toWorkspaceMemberJSON(item))
	}
	writeJSON(w, http.StatusOK, listWorkspaceMembersResponse{Members: out})
}

func writeWorkspaceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidHandle):
		writeError(w, http.StatusBadRequest, "invalid_handle", "Handle must be 2 to 32 characters using letters, numbers, and underscores")
	case errors.Is(err, service.ErrHandleTaken):
		writeError(w, http.StatusConflict, "handle_taken", "That handle is already taken in this workspace")
	case errors.Is(err, service.ErrInvalidIcon):
		writeError(w, http.StatusBadRequest, "invalid_icon", "Icon must be a PNG, JPEG, WebP, or GIF under 2 MB")
	case errors.Is(err, service.ErrWorkspaceName), errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", "Invalid workspace name")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Workspace not found")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "You do not have permission to manage this workspace")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Something went wrong")
	}
}
