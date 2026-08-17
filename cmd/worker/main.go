package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	"github.com/kilobyteno/dagr-chat/internal/billing"
	"github.com/kilobyteno/dagr-chat/internal/config"
	"github.com/kilobyteno/dagr-chat/internal/email"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
	"github.com/kilobyteno/dagr-chat/internal/service"
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
	notificationService := service.NewNotificationService(store)
	channelService := service.NewChannelService(store).WithNotifications(notificationService)
	messageService := service.NewMessageService(store, channelService).
		WithNotifications(notificationService, notificationService).
		WithEntitlements(billingService)

	mailSender, err := email.NewSender(cfg, logger)
	if err != nil {
		logger.Error("email sender unavailable", "error", err)
		os.Exit(1)
	}
	logger.Info("email sender ready", "provider", cfg.EmailProvider)

	redisOpt, err := worker.ParseRedisOpt(cfg.RedisURL)
	if err != nil {
		logger.Error("invalid redis url", "error", err)
		os.Exit(1)
	}

	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 4,
		Queues: map[string]int{
			"default": 10,
		},
	})

	mux := asynq.NewServeMux()
	handlers := &worker.Handlers{Messages: messageService, Billing: billingService, Mail: mailSender, Logger: logger}
	handlers.Register(mux)

	scheduler := asynq.NewScheduler(redisOpt, nil)
	if err := worker.SchedulePeriodicPublish(scheduler, 15*time.Second); err != nil {
		logger.Error("schedule publish task", "error", err)
		os.Exit(1)
	}
	if err := worker.SchedulePeriodicMaintenance(scheduler); err != nil {
		logger.Error("schedule maintenance tasks", "error", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("asynq scheduler starting")
		if err := scheduler.Run(); err != nil {
			logger.Error("scheduler failed", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		logger.Info("dagr worker listening", "redis", cfg.RedisURL, "tasks", worker.TaskTypes)
		if err := server.Run(mux); err != nil {
			logger.Error("worker failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down worker")
	scheduler.Shutdown()
	server.Shutdown()
}
