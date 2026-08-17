package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/kilobyteno/dagr-chat/internal/billing"
	"github.com/kilobyteno/dagr-chat/internal/config"
	"github.com/kilobyteno/dagr-chat/internal/presence"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
	"github.com/kilobyteno/dagr-chat/internal/service"
	httpserver "github.com/kilobyteno/dagr-chat/internal/transport/http"
	"github.com/kilobyteno/dagr-chat/internal/worker"
)

func main() {
	cfg := config.Load()
	logger := cfg.NewLogger()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	store := postgres.NewStore(pool)
	var mollieProvider billing.Provider
	if cfg.BillingEnabled() && strings.TrimSpace(cfg.MollieAPIKey) != "" {
		mollieProvider = &billing.Mollie{APIKey: cfg.MollieAPIKey}
	}
	billingService := service.NewBillingService(store, cfg, mollieProvider, logger)
	domainService := service.NewDomainService(store).WithLifecycle(billingService)
	workspaceService := service.NewWorkspaceService(store).WithLifecycle(billingService)
	notificationService := service.NewNotificationService(store)
	channelService := service.NewChannelService(store).WithNotifications(notificationService)
	messageService := service.NewMessageService(store, channelService).
		WithNotifications(notificationService, notificationService).
		WithEntitlements(billingService)

	var mailer *worker.AsynqMailer
	var presenceStore presence.Store = presence.NewMemory(0)
	redisOpt, err := worker.ParseRedisOpt(cfg.RedisURL, cfg.RedisTLS, cfg.RedisTLSSkipVerify)
	if err != nil {
		logger.Warn("asynq client disabled; invite and verification emails and link previews will not enqueue", "error", err)
	} else {
		asynqClient := asynq.NewClient(redisOpt)
		defer asynqClient.Close()
		mailer = &worker.AsynqMailer{Client: asynqClient}
		messageService = messageService.WithLinkUnfurl(&worker.AsynqLinkUnfurlEnqueuer{Client: asynqClient})
		redisClient := redis.NewClient(&redis.Options{
			Addr:      redisOpt.Addr,
			Username:  redisOpt.Username,
			Password:  redisOpt.Password,
			DB:        redisOpt.DB,
			TLSConfig: redisOpt.TLSConfig,
		})
		defer redisClient.Close()
		if pingErr := redisClient.Ping(context.Background()).Err(); pingErr != nil {
			logger.Warn("presence redis unavailable; using in-memory store", "error", pingErr)
		} else {
			presenceStore = presence.NewRedis(redisClient, 0)
		}
	}
	var inviteMailer service.InviteMailer
	var verificationMailer service.VerificationMailer
	if mailer != nil {
		inviteMailer = mailer
		verificationMailer = mailer
	}
	authService := service.NewAuthService(store, cfg.PasswordPolicy, cfg.SessionTTL).
		WithAutoJoiner(domainService).
		WithVerificationMailer(verificationMailer, cfg.PublicBaseURL)
	inviteService := service.NewInviteService(store, cfg.PublicBaseURL, inviteMailer).
		WithNotifications(notificationService).
		WithLifecycle(billingService)

	api := httpserver.NewServer(
		cfg, authService, workspaceService, domainService,
		channelService, inviteService, messageService, notificationService,
		presenceStore, logger,
	).WithBilling(billingService)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("dagr server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	logger.Info("shutting down")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}
