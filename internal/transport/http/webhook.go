package http

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kilobyteno/dagr-chat/internal/service"
)

const maxIncomingWebhookBytes = 64 << 10

func (s *Server) handleIncomingWebhook(w http.ResponseWriter, r *http.Request) {
	if s.webhooks == nil {
		s.writeError(w, r, http.StatusNotFound, "not_found", "Webhook was not found", nil)
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxIncomingWebhookBytes))
	if err != nil {
		s.writeError(w, r, http.StatusRequestEntityTooLarge, "invalid_input", "Webhook payload is too large", err)
		return
	}
	msg, err := s.webhooks.Receive(r.Context(), chi.URLParam(r, "token"), raw)
	if err != nil {
		s.writeIncomingWebhookError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": toMessageJSON(*msg)})
}

func (s *Server) writeIncomingWebhookError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		s.writeError(w, r, http.StatusNotFound, "not_found", "Webhook was not found", err)
	case errors.Is(err, service.ErrWebhookRateLimited):
		s.writeError(w, r, http.StatusTooManyRequests, "rate_limited", "Too many webhook posts. Try again in a minute", err)
	case errors.Is(err, service.ErrInvalidInput):
		s.writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid webhook payload", err)
	case errors.Is(err, service.ErrNotAChannel):
		s.writeError(w, r, http.StatusBadRequest, "invalid_input", "Incoming webhooks cannot post to direct messages", err)
	default:
		s.writeError(w, r, http.StatusInternalServerError, "internal_error", "Could not deliver webhook", err)
	}
}
