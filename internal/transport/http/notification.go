package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type notificationJSON struct {
	ID          string     `json:"id"`
	Kind        string     `json:"kind"`
	Body        string     `json:"body"`
	ActorID     string     `json:"actorId,omitempty"`
	ActorName   string     `json:"actorName,omitempty"`
	WorkspaceID string     `json:"workspaceId,omitempty"`
	ChannelID   string     `json:"channelId,omitempty"`
	ChannelName string     `json:"channelName,omitempty"`
	IsDM        bool       `json:"isDm,omitempty"`
	MessageID   string     `json:"messageId,omitempty"`
	Unread      bool       `json:"unread"`
	ReadAt      *time.Time `json:"readAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func toNotificationJSON(n domain.Notification) notificationJSON {
	return notificationJSON{
		ID: n.ID, Kind: string(n.Kind), Body: n.Body,
		ActorID: n.ActorID, ActorName: n.ActorName,
		WorkspaceID: n.WorkspaceID, ChannelID: n.ChannelID, ChannelName: n.ChannelName,
		IsDM: n.IsDM, MessageID: n.MessageID, Unread: n.ReadAt == nil, ReadAt: n.ReadAt,
		CreatedAt: n.CreatedAt,
	}
}

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	filter := r.URL.Query().Get("filter")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.notifications.List(r.Context(), user.ID, filter, limit)
	if err != nil {
		s.writeNotificationError(w, r, err)
		return
	}
	unread, err := s.notifications.UnreadCount(r.Context(), user.ID)
	if err != nil {
		s.writeNotificationError(w, r, err)
		return
	}
	out := make([]notificationJSON, 0, len(items))
	for _, item := range items {
		out = append(out, toNotificationJSON(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": out,
		"unreadCount":   unread,
	})
}

func (s *Server) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	if err := s.notifications.MarkRead(r.Context(), user.ID, chi.URLParam(r, "notificationID")); err != nil {
		s.writeNotificationError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	if err := s.notifications.MarkAllRead(r.Context(), user.ID); err != nil {
		s.writeNotificationError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type channelNotificationSettingsRequest struct {
	Level string `json:"level"`
}

func (s *Server) handleGetChannelNotificationSettings(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	channelID := chi.URLParam(r, "channelID")
	if _, err := s.channels.CanAccessChannel(r.Context(), user.ID, channelID); err != nil {
		s.writeChannelError(w, r, err)
		return
	}
	level, err := s.notifications.GetChannelNotificationLevel(r.Context(), user.ID, channelID)
	if err != nil {
		s.writeNotificationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"level": level})
}

func (s *Server) handlePutChannelNotificationSettings(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	channelID := chi.URLParam(r, "channelID")
	if _, err := s.channels.CanAccessChannel(r.Context(), user.ID, channelID); err != nil {
		s.writeChannelError(w, r, err)
		return
	}
	var req channelNotificationSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	level, err := s.notifications.SetChannelNotificationLevel(r.Context(), user.ID, channelID, req.Level)
	if err != nil {
		s.writeNotificationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"level": level})
}

func (s *Server) writeNotificationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		s.writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid notification request", err)
	case errors.Is(err, service.ErrNotFound):
		s.writeError(w, r, http.StatusNotFound, "not_found", "Notification not found", err)
	default:
		s.writeError(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong", err)
	}
}
