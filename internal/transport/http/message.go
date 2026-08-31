package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type postMessageRequest struct {
	Body string `json:"body"`
}

type channelReadRequest struct {
	MessageID string `json:"messageId"`
}

type channelUnreadJSON struct {
	UnreadCount          int    `json:"unreadCount"`
	FirstUnreadMessageID string `json:"firstUnreadMessageId,omitempty"`
}

type scheduleMessageRequest struct {
	Body   string    `json:"body"`
	SendAt time.Time `json:"sendAt"`
}

type linkPreviewJSON struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Status      string `json:"status"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	SiteName    string `json:"siteName,omitempty"`
	ImageURL    string `json:"imageUrl,omitempty"`
}

type reactionJSON struct {
	Emoji   string   `json:"emoji"`
	Count   int      `json:"count"`
	Reacted bool     `json:"reacted"`
	UserIDs []string `json:"userIds,omitempty"`
}

type messageJSON struct {
	ID                    string            `json:"id"`
	ChannelID             string            `json:"channelId"`
	AuthorID              string            `json:"authorId"`
	AuthorName            string            `json:"authorName,omitempty"`
	AuthorHandle          string            `json:"authorHandle,omitempty"`
	AuthorHasAvatar       bool                   `json:"authorHasAvatar"`
	AuthorAvatarUpdatedAt *time.Time             `json:"authorAvatarUpdatedAt,omitempty"`
	AuthorKind            string                 `json:"authorKind,omitempty"`
	AuthorIconURL         string                 `json:"authorIconUrl,omitempty"`
	Body                  string                 `json:"body"`
	ContentType           string                 `json:"contentType"`
	Payload               *domain.RichPayload    `json:"payload,omitempty"`
	LinkPreviews          []linkPreviewJSON      `json:"linkPreviews,omitempty"`
	Reactions             []reactionJSON         `json:"reactions,omitempty"`
	CreatedAt             time.Time              `json:"createdAt"`
	EditedAt              *time.Time             `json:"editedAt,omitempty"`
}

type reactionRequest struct {
	Emoji string `json:"emoji"`
}

type scheduledJSON struct {
	ID          string    `json:"id"`
	ChannelID   string    `json:"channelId"`
	AuthorID    string    `json:"authorId"`
	Body        string    `json:"body"`
	ContentType string    `json:"contentType"`
	SendAt      time.Time `json:"sendAt"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

func toMessageJSON(m domain.Message) messageJSON {
	out := messageJSON{
		ID: m.ID, ChannelID: m.ChannelID, AuthorID: m.AuthorID,
		AuthorName: m.AuthorName, AuthorHandle: m.AuthorHandle,
		AuthorHasAvatar: m.AuthorHasAvatar, AuthorAvatarUpdatedAt: m.AuthorAvatarUpdated,
		AuthorKind: m.AuthorKind, AuthorIconURL: m.AuthorIconURL,
		Body: m.Body, ContentType: m.ContentType, Payload: m.Payload, CreatedAt: m.CreatedAt,
	}
	if m.UpdatedAt.Sub(m.CreatedAt) > time.Second {
		editedAt := m.UpdatedAt.UTC()
		out.EditedAt = &editedAt
	}
	if len(m.LinkPreviews) > 0 {
		out.LinkPreviews = make([]linkPreviewJSON, 0, len(m.LinkPreviews))
		for _, preview := range m.LinkPreviews {
			out.LinkPreviews = append(out.LinkPreviews, linkPreviewJSON{
				ID: preview.ID, URL: preview.URL, Status: string(preview.Status),
				Title: preview.Title, Description: preview.Description,
				SiteName: preview.SiteName, ImageURL: preview.ImageURL,
			})
		}
	}
	if len(m.Reactions) > 0 {
		out.Reactions = make([]reactionJSON, 0, len(m.Reactions))
		for _, reaction := range m.Reactions {
			out.Reactions = append(out.Reactions, reactionJSON{
				Emoji: reaction.Emoji, Count: reaction.Count, Reacted: reaction.Reacted,
				UserIDs: reaction.UserIDs,
			})
		}
	}
	return out
}

func toScheduledJSON(m domain.ScheduledMessage) scheduledJSON {
	return scheduledJSON{
		ID: m.ID, ChannelID: m.ChannelID, AuthorID: m.AuthorID, Body: m.Body,
		ContentType: m.ContentType, SendAt: m.SendAt, Status: string(m.Status), CreatedAt: m.CreatedAt,
	}
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var before *time.Time
	var beforeID *uuid.UUID
	if raw := r.URL.Query().Get("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			t, err = time.Parse(time.RFC3339, raw)
		}
		if err != nil {
			s.writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid before cursor", nil)
			return
		}
		before = &t
	}
	if raw := r.URL.Query().Get("beforeId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			s.writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid beforeId", nil)
			return
		}
		beforeID = &id
	}
	result, err := s.messages.ListWithMeta(r.Context(), user.ID, chi.URLParam(r, "channelID"), before, beforeID, limit)
	if err != nil {
		s.writeMessageError(w, r, err)
		return
	}
	out := make([]messageJSON, 0, len(result.Messages))
	for _, item := range result.Messages {
		out = append(out, toMessageJSON(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"messages":             out,
		"historyLimited":       result.HistoryLimited,
		"historyRetentionDays": result.HistoryRetentionDays,
	})
}

func (s *Server) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	var req postMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	msg, err := s.messages.Post(r.Context(), user.ID, chi.URLParam(r, "channelID"), req.Body)
	if err != nil {
		s.writeMessageError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": toMessageJSON(*msg)})
}

func (s *Server) handleUpdateMessage(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	var req postMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	msg, err := s.messages.Update(r.Context(), user.ID, chi.URLParam(r, "messageID"), req.Body)
	if err != nil {
		s.writeMessageError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": toMessageJSON(*msg)})
}

func (s *Server) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	if err := s.messages.Delete(r.Context(), user.ID, chi.URLParam(r, "messageID")); err != nil {
		s.writeMessageError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleToggleMessageReaction(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	var req reactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	msg, err := s.messages.ToggleReaction(
		r.Context(), user.ID, chi.URLParam(r, "messageID"), req.Emoji,
	)
	if err != nil {
		s.writeMessageError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": toMessageJSON(*msg)})
}

func (s *Server) handleRemoveMessageReaction(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	msg, err := s.messages.RemoveReaction(
		r.Context(), user.ID, chi.URLParam(r, "messageID"), chi.URLParam(r, "emoji"),
	)
	if err != nil {
		s.writeMessageError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": toMessageJSON(*msg)})
}

func toChannelUnreadJSON(u service.ChannelUnread) channelUnreadJSON {
	return channelUnreadJSON{
		UnreadCount:          u.UnreadCount,
		FirstUnreadMessageID: u.FirstUnreadMessageID,
	}
}

func (s *Server) handleMarkChannelRead(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	var req channelReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	unread, err := s.messages.MarkRead(r.Context(), user.ID, chi.URLParam(r, "channelID"), req.MessageID)
	if err != nil {
		s.writeMessageError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toChannelUnreadJSON(*unread))
}

func (s *Server) handleMarkChannelUnread(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	var req channelReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	if strings.TrimSpace(req.MessageID) == "" {
		s.writeError(w, r, http.StatusBadRequest, "invalid_input", "messageId is required", nil)
		return
	}
	unread, err := s.messages.MarkUnread(r.Context(), user.ID, chi.URLParam(r, "channelID"), req.MessageID)
	if err != nil {
		s.writeMessageError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toChannelUnreadJSON(*unread))
}

func (s *Server) handleScheduleMessage(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	var req scheduleMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	msg, err := s.messages.Schedule(r.Context(), user.ID, chi.URLParam(r, "channelID"), req.Body, req.SendAt)
	if err != nil {
		s.writeMessageError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"scheduledMessage": toScheduledJSON(*msg)})
}

func (s *Server) handleListScheduledMessages(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	items, err := s.messages.ListScheduled(r.Context(), user.ID, chi.URLParam(r, "channelID"))
	if err != nil {
		s.writeMessageError(w, r, err)
		return
	}
	out := make([]scheduledJSON, 0, len(items))
	for _, item := range items {
		out = append(out, toScheduledJSON(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"scheduledMessages": out})
}

func (s *Server) handleCancelScheduledMessage(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	if err := s.messages.CancelScheduled(r.Context(), user.ID, chi.URLParam(r, "scheduledID")); err != nil {
		s.writeMessageError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) writeMessageError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		s.writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid message", err)
	case errors.Is(err, service.ErrNotFound):
		s.writeError(w, r, http.StatusNotFound, "not_found", "Not found", err)
	case errors.Is(err, service.ErrForbidden):
		s.writeError(w, r, http.StatusForbidden, "forbidden", "You do not have permission", err)
	default:
		s.writeError(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong", err)
	}
}
