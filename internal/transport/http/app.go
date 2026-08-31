package http

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type appJSON struct {
	Slug         string   `json:"slug"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Origin       string   `json:"origin"`
	Capabilities []string `json:"capabilities"`
	Installed    bool     `json:"installed"`
	Channels     []incomingWebhookJSON `json:"channels,omitempty"`
}

type incomingWebhookJSON struct {
	ID               string  `json:"id"`
	ChannelID        string  `json:"channelId"`
	ChannelName      string  `json:"channelName,omitempty"`
	ChannelIsPrivate bool    `json:"channelIsPrivate,omitempty"`
	TokenPrefix      string  `json:"tokenPrefix"`
	URL              string  `json:"url,omitempty"`
	LastUsedAt       *string `json:"lastUsedAt,omitempty"`
	CreatedAt        string  `json:"createdAt"`
}

type listAppsResponse struct {
	Apps []appJSON `json:"apps"`
}

func toIncomingWebhookJSON(hook domain.IncomingWebhook) incomingWebhookJSON {
	out := incomingWebhookJSON{
		ID: hook.ID, ChannelID: hook.ChannelID, ChannelName: hook.ChannelName,
		ChannelIsPrivate: hook.ChannelIsPrivate, TokenPrefix: hook.TokenPrefix,
		URL: hook.URL, CreatedAt: hook.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000000Z07:00"),
	}
	if hook.LastUsedAt != nil {
		stamp := hook.LastUsedAt.UTC().Format("2006-01-02T15:04:05.000000000Z07:00")
		out.LastUsedAt = &stamp
	}
	return out
}

func toAppJSON(item domain.WorkspaceApp) appJSON {
	out := appJSON{
		Slug: item.App.Slug, Name: item.App.Name, Description: item.App.Description,
		Origin: item.App.Origin, Capabilities: item.App.Capabilities, Installed: item.Installed,
	}
	if out.Capabilities == nil {
		out.Capabilities = []string{}
	}
	if len(item.ChannelHooks) > 0 {
		out.Channels = make([]incomingWebhookJSON, 0, len(item.ChannelHooks))
		for _, hook := range item.ChannelHooks {
			out.Channels = append(out.Channels, toIncomingWebhookJSON(hook))
		}
	}
	return out
}

func (s *Server) handleListWorkspaceApps(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.apps == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	items, err := s.apps.ListForWorkspace(r.Context(), user.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		s.writeAppError(w, r, err)
		return
	}
	out := make([]appJSON, 0, len(items))
	for _, item := range items {
		out = append(out, toAppJSON(item))
	}
	writeJSON(w, http.StatusOK, listAppsResponse{Apps: out})
}

func (s *Server) handleInstallWorkspaceApp(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.apps == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	install, err := s.apps.Install(r.Context(), user.ID, chi.URLParam(r, "workspaceID"), chi.URLParam(r, "appSlug"))
	if err != nil {
		s.writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"app": map[string]string{
			"slug": install.AppSlug,
			"name": install.AppName,
			"id":   install.ID,
		},
	})
}

func (s *Server) handleUninstallWorkspaceApp(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.apps == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	if err := s.apps.Uninstall(r.Context(), user.ID, chi.URLParam(r, "workspaceID"), chi.URLParam(r, "appSlug")); err != nil {
		s.writeAppError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetChannelIncomingWebhook(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.webhooks == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	hook, err := s.webhooks.GetForChannel(r.Context(), user.ID, chi.URLParam(r, "channelID"))
	if err != nil {
		s.writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"webhook": toIncomingWebhookJSON(*hook)})
}

func (s *Server) handleEnableChannelIncomingWebhook(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.webhooks == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	secret, err := s.webhooks.EnableOnChannel(r.Context(), user.ID, chi.URLParam(r, "channelID"))
	if err != nil {
		s.writeAppError(w, r, err)
		return
	}
	status := http.StatusCreated
	if secret.Token == "" {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{"webhook": toIncomingWebhookJSON(secret.IncomingWebhook)})
}

func (s *Server) handleRotateChannelIncomingWebhook(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.webhooks == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	secret, err := s.webhooks.RotateOnChannel(r.Context(), user.ID, chi.URLParam(r, "channelID"))
	if err != nil {
		s.writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"webhook": toIncomingWebhookJSON(secret.IncomingWebhook)})
}

func (s *Server) handleDisableChannelIncomingWebhook(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || s.webhooks == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	if err := s.webhooks.DisableOnChannel(r.Context(), user.ID, chi.URLParam(r, "channelID")); err != nil {
		s.writeAppError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) writeAppError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		s.writeError(w, r, http.StatusForbidden, "forbidden", "You do not have permission to manage apps", err)
	case errors.Is(err, service.ErrNotFound):
		s.writeError(w, r, http.StatusNotFound, "not_found", "App or webhook was not found", err)
	case errors.Is(err, service.ErrNotAChannel):
		s.writeError(w, r, http.StatusBadRequest, "invalid_input", "Incoming webhooks cannot be added to direct messages", err)
	case errors.Is(err, service.ErrAppLimit):
		s.writeError(w, r, http.StatusForbidden, "app_limit", "This workspace has reached its app limit", err)
	case errors.Is(err, service.ErrAlreadyInstalled):
		s.writeError(w, r, http.StatusConflict, "already_installed", "This app is already installed", err)
	case errors.Is(err, service.ErrInvalidInput):
		s.writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid app request", err)
	default:
		s.writeError(w, r, http.StatusInternalServerError, "internal_error", "Could not manage apps", err)
	}
}
