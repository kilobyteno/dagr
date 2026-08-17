package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type contextKey string

const userContextKey contextKey = "user"
const requestLogStateKey contextKey = "requestLogState"

type requestLogState struct {
	userID string
}

func UserFromContext(ctx context.Context) *domain.User {
	user, _ := ctx.Value(userContextKey).(*domain.User)
	return user
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		user, _, err := s.auth.Authenticate(r.Context(), token)
		if err != nil {
			if errors.Is(err, service.ErrUnauthorized) {
				s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", err)
				return
			}
			s.writeError(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong", err)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		if state, ok := r.Context().Value(requestLogStateKey).(*requestLogState); ok && state != nil {
			state.userID = user.ID
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		state := &requestLogState{}
		r = r.WithContext(context.WithValue(r.Context(), requestLogStateKey, state))
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		reqID := middleware.GetReqID(r.Context())
		if reqID != "" {
			ww.Header().Set(middleware.RequestIDHeader, reqID)
		}
		next.ServeHTTP(ww, r)

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"durationMs", time.Since(start).Milliseconds(),
			"requestId", reqID,
		}
		if state.userID != "" {
			attrs = append(attrs, "userId", state.userID)
		}
		if r.URL.Path == "/api/v1/health" {
			s.logger.Debug("http request", attrs...)
			return
		}
		s.logger.Info("http request", attrs...)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			s.logger.Error("http panic",
				"error", fmt.Errorf("panic: %v", rec),
				"requestId", middleware.GetReqID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
			)
			writeJSON(w, http.StatusInternalServerError, apiErrorBody{
				Error: apiError{Code: "internal_error", Message: "Something went wrong"},
			})
		}()
		next.ServeHTTP(w, r)
	})
}
