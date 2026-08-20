package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type signupRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type publicUser struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	DisplayName       string     `json:"displayName"`
	NotificationLevel string     `json:"notificationLevel"`
	Locale            string     `json:"locale"`
	EmailVerified     bool       `json:"emailVerified"`
	StatusEmoji       string     `json:"statusEmoji,omitempty"`
	StatusText        string     `json:"statusText,omitempty"`
	StatusExpiresAt   *time.Time `json:"statusExpiresAt,omitempty"`
	HasAvatar         bool       `json:"hasAvatar"`
	AvatarUpdatedAt   *time.Time `json:"avatarUpdatedAt,omitempty"`
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

type authResponse struct {
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expiresAt"`
	User      publicUser `json:"user"`
}

type meResponse struct {
	User publicUser `json:"user"`
}

type updateProfileRequest struct {
	DisplayName       string `json:"displayName"`
	NotificationLevel string `json:"notificationLevel"`
	Locale            string `json:"locale"`
}

type updateStatusRequest struct {
	Emoji     string     `json:"emoji"`
	Text      string     `json:"text"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

type updatePresenceRequest struct {
	State string `json:"state"`
}

func toPublicUser(u domain.User) publicUser {
	level := u.NotificationLevel
	if level == "" {
		level = domain.NotifyMentions
	}
	locale := u.Locale
	if locale == "" {
		locale = domain.DefaultLocale()
	}
	return publicUser{
		ID:                u.ID,
		Email:             u.Email,
		DisplayName:       u.DisplayName,
		NotificationLevel: string(level),
		Locale:            string(locale),
		EmailVerified:     u.EmailVerified,
		StatusEmoji:       u.StatusEmoji,
		StatusText:        u.StatusText,
		StatusExpiresAt:   u.StatusExpiresAt,
		HasAvatar:         u.HasAvatar,
		AvatarUpdatedAt:   u.AvatarUpdatedAt,
	}
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}

	result, err := s.auth.Signup(r.Context(), service.SignupInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		s.writeAuthError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt.UTC(),
		User:      toPublicUser(result.User),
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}

	result, err := s.auth.Login(r.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		s.writeAuthError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt.UTC(),
		User:      toPublicUser(result.User),
	})
}

func (s *Server) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	user, err := s.auth.VerifyEmail(r.Context(), req.Token)
	if err != nil {
		s.writeAuthError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, meResponse{User: toPublicUser(*user)})
}

func (s *Server) handleResendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	if err := s.auth.ResendVerificationEmail(r.Context(), user.ID); err != nil {
		s.writeAuthError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	if user := UserFromContext(r.Context()); user != nil && s.presence != nil {
		_ = s.presence.Clear(r.Context(), user.ID)
	}
	if err := s.auth.Logout(r.Context(), token); err != nil {
		s.writeAuthError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	writeJSON(w, http.StatusOK, meResponse{User: toPublicUser(*user)})
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	updated, err := s.auth.UpdateProfile(r.Context(), user.ID, req.DisplayName, req.NotificationLevel, req.Locale)
	if err != nil {
		s.writeAuthError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, meResponse{User: toPublicUser(*updated)})
}

func (s *Server) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	updated, err := s.auth.UpdateStatus(r.Context(), user.ID, req.Emoji, req.Text, req.ExpiresAt)
	if err != nil {
		s.writeAuthError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, meResponse{User: toPublicUser(*updated)})
}

func (s *Server) handleUpdatePresence(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	var req updatePresenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	state := strings.TrimSpace(strings.ToLower(req.State))
	switch state {
	case "active", "online":
		if s.presence != nil {
			_ = s.presence.Touch(r.Context(), user.ID, false)
		}
		writeJSON(w, http.StatusOK, map[string]string{"presence": "online"})
	case "away":
		if s.presence != nil {
			_ = s.presence.Touch(r.Context(), user.ID, true)
		}
		writeJSON(w, http.StatusOK, map[string]string{"presence": "away"})
	case "offline":
		if s.presence != nil {
			_ = s.presence.Clear(r.Context(), user.ID)
		}
		writeJSON(w, http.StatusOK, map[string]string{"presence": "offline"})
	default:
		s.writeError(w, r, http.StatusBadRequest, "invalid_input", "Presence state must be active, away, or offline", nil)
	}
}

func (s *Server) handlePutMyAvatar(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	const maxMemory = 2 << 20
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_avatar", "Expected a multipart image upload", nil)
		return
	}
	file, header, err := r.FormFile("avatar")
	if err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_avatar", "Missing avatar file", nil)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	updated, err := s.auth.SetAvatar(r.Context(), user.ID, file, contentType)
	if err != nil {
		s.writeAuthError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, meResponse{User: toPublicUser(*updated)})
}

func (s *Server) handleDeleteMyAvatar(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	updated, err := s.auth.ClearAvatar(r.Context(), user.ID)
	if err != nil {
		s.writeAuthError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, meResponse{User: toPublicUser(*updated)})
}

func (s *Server) handleGetMyAvatar(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	s.writeUserAvatar(w, r, user.ID)
}

func (s *Server) handleGetUserAvatar(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	s.writeUserAvatar(w, r, chi.URLParam(r, "userID"))
}

func (s *Server) writeUserAvatar(w http.ResponseWriter, r *http.Request, userID string) {
	avatar, err := s.auth.GetAvatar(r.Context(), userID)
	if err != nil {
		s.writeAuthError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", avatar.ContentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	if !avatar.UpdatedAt.IsZero() {
		w.Header().Set("Last-Modified", avatar.UpdatedAt.UTC().Format(http.TimeFormat))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, bytes.NewReader(avatar.Bytes))
}

func (s *Server) writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		s.writeError(w, r, http.StatusBadRequest, "invalid_input", "Email, password, and display name are required", err)
	case errors.Is(err, service.ErrWeakPassword):
		msg := err.Error()
		if i := strings.Index(msg, ": "); i >= 0 {
			msg = msg[i+2:]
		}
		s.writeError(w, r, http.StatusBadRequest, "weak_password", msg, err)
	case errors.Is(err, service.ErrEmailTaken):
		s.writeError(w, r, http.StatusConflict, "email_taken", "An account with this email already exists", err)
	case errors.Is(err, service.ErrInvalidCredentials):
		s.writeError(w, r, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password", err)
	case errors.Is(err, service.ErrUnauthorized):
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", err)
	case errors.Is(err, service.ErrInvalidAvatar):
		s.writeError(w, r, http.StatusBadRequest, "invalid_avatar", "Profile picture must be a PNG, JPEG, WebP, or GIF under 2 MB", err)
	case errors.Is(err, service.ErrInvalidVerificationToken):
		s.writeError(w, r, http.StatusBadRequest, "invalid_verification_token", "This verification link is invalid or has expired", err)
	case errors.Is(err, service.ErrVerificationRateLimited):
		s.writeError(w, r, http.StatusTooManyRequests, "verification_rate_limited", "Please wait a minute before requesting another verification email", err)
	case errors.Is(err, service.ErrNotFound):
		s.writeError(w, r, http.StatusNotFound, "not_found", "Not found", err)
	default:
		s.writeError(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong", err)
	}
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
