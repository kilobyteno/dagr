package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

type apiErrorBody struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, status int, code, message string, cause error) {
	s.logAPIError(r, status, code, cause)
	writeJSON(w, status, apiErrorBody{
		Error: apiError{Code: code, Message: message},
	})
}

func (s *Server) logAPIError(r *http.Request, status int, code string, cause error) {
	if s == nil || s.logger == nil || r == nil {
		return
	}
	attrs := []any{
		"code", code,
		"status", status,
		"method", r.Method,
		"path", r.URL.Path,
		"requestId", middleware.GetReqID(r.Context()),
	}
	if user := UserFromContext(r.Context()); user != nil {
		attrs = append(attrs, "userId", user.ID)
	}
	if cause != nil {
		attrs = append(attrs, "error", cause)
	}
	if status >= 500 {
		s.logger.Error("api error", attrs...)
		return
	}
	s.logger.Info("api error", attrs...)
}
