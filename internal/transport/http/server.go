package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/kilobyteno/dagr-chat/internal/auth"
	"github.com/kilobyteno/dagr-chat/internal/billing"
	"github.com/kilobyteno/dagr-chat/internal/config"
	"github.com/kilobyteno/dagr-chat/internal/presence"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

// Server hosts the versioned HTTP API.
type Server struct {
	cfg           config.Config
	auth          *service.AuthService
	workspaces    *service.WorkspaceService
	domains       *service.DomainService
	channels      *service.ChannelService
	invites       *service.InviteService
	messages      *service.MessageService
	notifications *service.NotificationService
	presence      presence.Store
	billing       *service.BillingService
	logger        *slog.Logger
}

// NewServer constructs the API server with its dependencies.
func NewServer(
	cfg config.Config,
	authService *service.AuthService,
	workspaceService *service.WorkspaceService,
	domainService *service.DomainService,
	channelService *service.ChannelService,
	inviteService *service.InviteService,
	messageService *service.MessageService,
	notificationService *service.NotificationService,
	presenceStore presence.Store,
	logger *slog.Logger,
) *Server {
	if presenceStore == nil {
		presenceStore = presence.NewMemory(0)
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		cfg:           cfg,
		auth:          authService,
		workspaces:    workspaceService,
		domains:       domainService,
		channels:      channelService,
		invites:       inviteService,
		messages:      messageService,
		notifications: notificationService,
		presence:      presenceStore,
		logger:        logger,
	}
}

func (s *Server) WithBilling(billingService *service.BillingService) *Server {
	s.billing = billingService
	return s
}

// Handler returns the chi router.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(s.recoverer)
	r.Use(s.requestLogger)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(corsMiddleware)
	r.NotFound(s.handleNotFound)

	r.Get("/verify-email", s.handleVerifyEmailPage)
	r.Post("/verify-email", s.handleVerifyEmailForm)
	r.Get("/invites/accept", s.handleAcceptInvitePage)

	r.Route("/api/v1", func(r chi.Router) {
		r.NotFound(s.handleNotFound)
		r.Get("/health", handleHealth)
		r.Get("/public/config", s.handlePublicConfig)
		r.Post("/billing/webhooks/mollie", s.handleMollieWebhook)

		r.Post("/auth/signup", s.handleSignup)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/verify-email", s.handleVerifyEmail)

		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Post("/auth/logout", s.handleLogout)
			r.Get("/me", s.handleMe)
			r.Post("/me/email/resend-verification", s.handleResendVerificationEmail)
			r.Patch("/me", s.handleUpdateProfile)
			r.Put("/me/status", s.handleUpdateStatus)
			r.Post("/me/presence", s.handleUpdatePresence)
			r.Get("/me/avatar", s.handleGetMyAvatar)
			r.Put("/me/avatar", s.handlePutMyAvatar)
			r.Delete("/me/avatar", s.handleDeleteMyAvatar)
			r.Get("/users/{userID}/avatar", s.handleGetUserAvatar)

			r.Get("/workspaces", s.handleListWorkspaces)
			r.Post("/workspaces", s.handleCreateWorkspace)
			r.Get("/workspaces/{workspaceID}", s.handleGetWorkspace)
			r.Patch("/workspaces/{workspaceID}", s.handleRenameWorkspace)
			r.Delete("/workspaces/{workspaceID}", s.handleDeleteWorkspace)
			r.Get("/workspaces/{workspaceID}/icon", s.handleGetWorkspaceIcon)
			r.Put("/workspaces/{workspaceID}/icon", s.handlePutWorkspaceIcon)
			r.Delete("/workspaces/{workspaceID}/icon", s.handleDeleteWorkspaceIcon)
			r.Get("/workspaces/{workspaceID}/me", s.handleGetWorkspaceMe)
			r.Patch("/workspaces/{workspaceID}/me", s.handleUpdateWorkspaceMe)
			r.Get("/workspaces/{workspaceID}/members", s.handleListWorkspaceMembers)
			r.Patch("/workspaces/{workspaceID}/members/{userID}", s.handleUpdateWorkspaceMemberRole)
			r.Delete("/workspaces/{workspaceID}/members/{userID}", s.handleRemoveWorkspaceMember)
			r.Post("/workspaces/{workspaceID}/leave", s.handleLeaveWorkspace)
			r.Post("/workspaces/{workspaceID}/transfer-ownership", s.handleTransferWorkspaceOwnership)
			r.Get("/workspaces/{workspaceID}/channels", s.handleListChannels)
			r.Post("/workspaces/{workspaceID}/channels", s.handleCreateChannel)
			r.Post("/workspaces/{workspaceID}/dms", s.handleOpenDM)
			r.Patch("/channels/{channelID}", s.handleUpdateChannel)
			r.Delete("/channels/{channelID}", s.handleDeleteChannel)
			r.Get("/channels/{channelID}/members", s.handleListChannelMembers)
			r.Post("/channels/{channelID}/members", s.handleAddChannelMember)
			r.Delete("/channels/{channelID}/members/{userID}", s.handleRemoveChannelMember)
			r.Get("/channels/{channelID}/notification-settings", s.handleGetChannelNotificationSettings)
			r.Put("/channels/{channelID}/notification-settings", s.handlePutChannelNotificationSettings)

			r.Post("/workspaces/{workspaceID}/invites", s.handleInviteToWorkspace)
			r.Get("/workspaces/{workspaceID}/invites", s.handleListInvites)
			r.Delete("/workspaces/{workspaceID}/invites/{inviteID}", s.handleRevokeInvite)
			r.Post("/invites/{token}/accept", s.handleAcceptInvite)

			r.Get("/channels/{channelID}/messages", s.handleListMessages)
			r.Post("/channels/{channelID}/messages", s.handlePostMessage)
			r.Post("/channels/{channelID}/read", s.handleMarkChannelRead)
			r.Post("/channels/{channelID}/unread", s.handleMarkChannelUnread)
			r.Patch("/messages/{messageID}", s.handleUpdateMessage)
			r.Delete("/messages/{messageID}", s.handleDeleteMessage)
			r.Post("/messages/{messageID}/reactions", s.handleToggleMessageReaction)
			r.Delete("/messages/{messageID}/reactions/{emoji}", s.handleRemoveMessageReaction)
			r.Post("/channels/{channelID}/scheduled-messages", s.handleScheduleMessage)
			r.Get("/channels/{channelID}/scheduled-messages", s.handleListScheduledMessages)
			r.Delete("/scheduled-messages/{scheduledID}", s.handleCancelScheduledMessage)

			r.Get("/notifications", s.handleListNotifications)
			r.Post("/notifications/read-all", s.handleMarkAllNotificationsRead)
			r.Post("/notifications/{notificationID}/read", s.handleMarkNotificationRead)

			r.Get("/workspaces/{workspaceID}/domains", s.handleListDomains)
			r.Post("/workspaces/{workspaceID}/domains", s.handleAddDomain)
			r.Post("/workspaces/{workspaceID}/domains/{domainID}/verify", s.handleVerifyDomain)
			r.Patch("/workspaces/{workspaceID}/domains/{domainID}", s.handlePatchDomain)
			r.Delete("/workspaces/{workspaceID}/domains/{domainID}", s.handleDeleteDomain)

			r.Get("/workspaces/{workspaceID}/billing", s.handleGetWorkspaceBilling)
			r.Post("/workspaces/{workspaceID}/billing/checkout", s.handleBillingCheckout)
			r.Post("/workspaces/{workspaceID}/billing/cancel", s.handleBillingCancel)
			r.Post("/workspaces/{workspaceID}/billing/resume", s.handleBillingResume)
		})
	})

	return r
}

// NewRouter builds the chi router with /api/v1 routes.
func NewRouter(
	cfg config.Config,
	authService *service.AuthService,
	workspaceService *service.WorkspaceService,
	domainService *service.DomainService,
	channelService *service.ChannelService,
	inviteService *service.InviteService,
	messageService *service.MessageService,
	notificationService *service.NotificationService,
	presenceStore presence.Store,
	logger *slog.Logger,
) http.Handler {
	return NewServer(
		cfg, authService, workspaceService, domainService,
		channelService, inviteService, messageService, notificationService,
		presenceStore, logger,
	).Handler()
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-Id")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-Id")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type publicConfigResponse struct {
	PasswordPolicy auth.PasswordPolicy  `json:"passwordPolicy"`
	DeploymentMode string               `json:"deploymentMode"`
	BillingEnabled bool                 `json:"billingEnabled"`
	Plans          *billing.PublicPlans `json:"plans,omitempty"`
}

func (s *Server) handlePublicConfig(w http.ResponseWriter, r *http.Request) {
	resp := publicConfigResponse{
		PasswordPolicy: s.cfg.PasswordPolicy,
		DeploymentMode: string(s.cfg.DeploymentMode),
		BillingEnabled: s.cfg.BillingEnabled(),
	}
	if resp.DeploymentMode == "" {
		resp.DeploymentMode = string(config.DeploymentSelfHosted)
	}
	if resp.BillingEnabled {
		catalog := billing.Catalog(s.cfg)
		resp.Plans = &catalog
	}
	writeJSON(w, http.StatusOK, resp)
}
