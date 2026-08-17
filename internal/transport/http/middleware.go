package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type contextKey string

const userContextKey contextKey = "user"

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
				writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "Something went wrong")
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
