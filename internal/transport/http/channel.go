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

type createChannelRequest struct {
	Name      string `json:"name"`
	Topic     string `json:"topic"`
	IsPrivate bool   `json:"isPrivate"`
}

type updateChannelRequest struct {
	Name      string `json:"name"`
	Topic     string `json:"topic"`
	IsPrivate bool   `json:"isPrivate"`
}

type channelMemberRequest struct {
	Email string `json:"email"`
}

type channelJSON struct {
	ID                   string     `json:"id"`
	WorkspaceID          string     `json:"workspaceId"`
	Name                 string     `json:"name"`
	Topic                string     `json:"topic,omitempty"`
	IsPrivate            bool       `json:"isPrivate"`
	IsDM                 bool       `json:"isDm,omitempty"`
	IsShared             bool       `json:"isShared,omitempty"`
	CreatedBy            string     `json:"createdBy,omitempty"`
	UnreadCount          int        `json:"unreadCount"`
	FirstUnreadMessageID string     `json:"firstUnreadMessageId,omitempty"`
	PeerUserID           string     `json:"peerUserId,omitempty"`
	PeerDisplayName      string     `json:"peerDisplayName,omitempty"`
	PeerHandle           string     `json:"peerHandle,omitempty"`
	PeerHasAvatar        bool       `json:"peerHasAvatar,omitempty"`
	PeerAvatarUpdatedAt  *time.Time `json:"peerAvatarUpdatedAt,omitempty"`
}

type openDMRequest struct {
	UserID string `json:"userId"`
}

func toChannelJSONFull(c domain.Channel) channelJSON {
	return channelJSON{
		ID:                   c.ID,
		WorkspaceID:          c.WorkspaceID,
		Name:                 c.Name,
		Topic:                c.Topic,
		IsPrivate:            c.IsPrivate,
		IsDM:                 c.IsDM,
		IsShared:             c.IsShared,
		CreatedBy:            c.CreatedBy,
		UnreadCount:          c.UnreadCount,
		FirstUnreadMessageID: c.FirstUnreadMessageID,
		PeerUserID:           c.PeerUserID,
		PeerDisplayName:      c.PeerDisplayName,
		PeerHandle:           c.PeerHandle,
		PeerHasAvatar:        c.PeerHasAvatar,
		PeerAvatarUpdatedAt:  c.PeerAvatarUpdatedAt,
	}
}

func (s *Server) handleOpenDM(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	var req openDMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	ch, err := s.channels.OpenDM(
		r.Context(), user.ID, chi.URLParam(r, "workspaceID"), req.UserID,
	)
	if err != nil {
		writeChannelError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"channel": toChannelJSONFull(*ch)})
}

func (s *Server) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	var req createChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	ch, err := s.channels.Create(r.Context(), user.ID, chi.URLParam(r, "workspaceID"), req.Name, req.Topic, req.IsPrivate)
	if err != nil {
		writeChannelError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"channel": toChannelJSONFull(*ch)})
}

func (s *Server) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	var req updateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	ch, err := s.channels.Update(
		r.Context(), user.ID, chi.URLParam(r, "channelID"), req.Name, req.Topic, req.IsPrivate,
	)
	if err != nil {
		writeChannelError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"channel": toChannelJSONFull(*ch)})
}

func (s *Server) handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	if err := s.channels.Delete(r.Context(), user.ID, chi.URLParam(r, "channelID")); err != nil {
		writeChannelError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListChannelMembers(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	items, err := s.channels.ListMembers(r.Context(), user.ID, chi.URLParam(r, "channelID"))
	if err != nil {
		writeChannelError(w, err)
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

func (s *Server) handleAddChannelMember(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	var req channelMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	if err := s.channels.AddMember(r.Context(), user.ID, chi.URLParam(r, "channelID"), req.Email); err != nil {
		writeChannelError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRemoveChannelMember(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	if err := s.channels.RemoveMember(
		r.Context(), user.ID, chi.URLParam(r, "channelID"), chi.URLParam(r, "userID"),
	); err != nil {
		writeChannelError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeChannelError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrChannelName), errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", "Invalid channel input")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Channel not found")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "You do not have permission")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Something went wrong")
	}
}
