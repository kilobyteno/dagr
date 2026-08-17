// Package worker defines asynq task types and handlers for background jobs.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/kilobyteno/dagr-chat/internal/email"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

const (
	// TaskTypeSendEmail is a placeholder for transactional email delivery.
	TaskTypeSendEmail = "email:send"
	// TaskTypeProcessAttachment is a placeholder for post-upload processing.
	TaskTypeProcessAttachment = "attachment:process"
	// TaskTypePublishScheduled publishes due scheduled messages.
	TaskTypePublishScheduled = "message:publish_scheduled"
	// TaskTypeUnfurlLink fetches Open Graph metadata for a message URL.
	TaskTypeUnfurlLink = "link:unfurl"
	// TaskTypePurgeExpired deletes Cloud Free messages older than the retention window.
	TaskTypePurgeExpired = "message:purge_expired"
	// TaskTypeReconcileBilling applies period-end downgrades and early-access price changes.
	TaskTypeReconcileBilling = "billing:reconcile"
)

// Task names used by the asynq client and server.
var TaskTypes = []string{
	TaskTypeSendEmail,
	TaskTypeProcessAttachment,
	TaskTypePublishScheduled,
	TaskTypeUnfurlLink,
	TaskTypePurgeExpired,
	TaskTypeReconcileBilling,
}

// EmailPayload is the payload for email:send.
type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// NewSendEmailTask builds an email:send task.
func NewSendEmailTask(to, subject, body string) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailPayload{To: to, Subject: subject, Body: body})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskTypeSendEmail, payload), nil
}

// NewPublishScheduledTask builds a message:publish_scheduled task.
func NewPublishScheduledTask() *asynq.Task {
	return asynq.NewTask(TaskTypePublishScheduled, nil)
}

// UnfurlLinkPayload identifies a pending message_link_previews row.
type UnfurlLinkPayload struct {
	PreviewID string `json:"previewId"`
}

// NewUnfurlLinkTask builds a link:unfurl task.
func NewUnfurlLinkTask(previewID string) (*asynq.Task, error) {
	payload, err := json.Marshal(UnfurlLinkPayload{PreviewID: previewID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskTypeUnfurlLink, payload), nil
}

// AsynqLinkUnfurlEnqueuer schedules link preview fetches.
type AsynqLinkUnfurlEnqueuer struct {
	Client *asynq.Client
}

// EnqueueLinkUnfurl implements service.LinkUnfurlEnqueuer.
func (e *AsynqLinkUnfurlEnqueuer) EnqueueLinkUnfurl(_ context.Context, previewID string) error {
	if e == nil || e.Client == nil || previewID == "" {
		return nil
	}
	task, err := NewUnfurlLinkTask(previewID)
	if err != nil {
		return err
	}
	_, err = e.Client.Enqueue(task, asynq.MaxRetry(3), asynq.Timeout(20*time.Second))
	return err
}

// ParseRedisOpt converts a Redis URL into asynq Redis client options.
func ParseRedisOpt(redisURL string) (asynq.RedisClientOpt, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return asynq.RedisClientOpt{}, fmt.Errorf("parse redis url: %w", err)
	}
	return asynq.RedisClientOpt{
		Addr:     opts.Addr,
		Username: opts.Username,
		Password: opts.Password,
		DB:       opts.DB,
	}, nil
}

// AsynqMailer enqueues transactional emails via asynq.
type AsynqMailer struct {
	Client *asynq.Client
}

// EnqueueInviteEmail implements service.InviteMailer.
func (m *AsynqMailer) EnqueueInviteEmail(_ context.Context, to, workspaceName, acceptURL string) error {
	if m == nil || m.Client == nil {
		return nil
	}
	task, err := NewSendEmailTask(
		to,
		fmt.Sprintf("You are invited to %s on Dagr", workspaceName),
		fmt.Sprintf("Accept your invite: %s", acceptURL),
	)
	if err != nil {
		return err
	}
	_, err = m.Client.Enqueue(task)
	return err
}

// EnqueueVerificationEmail implements service.VerificationMailer.
func (m *AsynqMailer) EnqueueVerificationEmail(_ context.Context, to, verifyURL string) error {
	if m == nil || m.Client == nil {
		return nil
	}
	task, err := NewSendEmailTask(
		to,
		"Verify your email for Dagr",
		fmt.Sprintf("Verify your email address by opening this link:\n\n%s\n\nIf you did not create a Dagr account, you can ignore this message.", verifyURL),
	)
	if err != nil {
		return err
	}
	_, err = m.Client.Enqueue(task)
	return err
}

// NewPurgeExpiredTask builds a message:purge_expired task.
func NewPurgeExpiredTask() *asynq.Task {
	return asynq.NewTask(TaskTypePurgeExpired, nil)
}

// NewReconcileBillingTask builds a billing:reconcile task.
func NewReconcileBillingTask() *asynq.Task {
	return asynq.NewTask(TaskTypeReconcileBilling, nil)
}

// Handlers registers asynq task handlers.
type Handlers struct {
	Messages *service.MessageService
	Billing  *service.BillingService
	Mail     email.Sender
	Logger   *slog.Logger
}

// Register attaches handlers to the mux.
func (h *Handlers) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskTypeSendEmail, h.handleSendEmail)
	mux.HandleFunc(TaskTypePublishScheduled, h.handlePublishScheduled)
	mux.HandleFunc(TaskTypeUnfurlLink, h.handleUnfurlLink)
	mux.HandleFunc(TaskTypePurgeExpired, h.handlePurgeExpired)
	mux.HandleFunc(TaskTypeReconcileBilling, h.handleReconcileBilling)
}

func (h *Handlers) handleSendEmail(ctx context.Context, task *asynq.Task) error {
	var payload EmailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		err = fmt.Errorf("decode email payload: %w", err)
		h.Logger.Error("email:send failed", "error", err)
		return err
	}
	if h.Mail == nil {
		h.Logger.Info("email:send (no sender)",
			"to", payload.To,
			"subject", payload.Subject,
			"bodyLen", len(payload.Body),
		)
		return nil
	}
	if err := h.Mail.Send(ctx, email.Message{
		To:      payload.To,
		Subject: payload.Subject,
		Text:    payload.Body,
	}); err != nil {
		h.Logger.Error("email:send failed", "error", err, "to", payload.To)
		return err
	}
	return nil
}

func (h *Handlers) handlePublishScheduled(ctx context.Context, _ *asynq.Task) error {
	if h.Messages == nil {
		return nil
	}
	n, err := h.Messages.PublishDue(ctx, time.Now().UTC(), 50)
	if err != nil {
		h.Logger.Error("message:publish_scheduled failed", "error", err)
		return err
	}
	if n > 0 {
		h.Logger.Info("published scheduled messages", "count", n)
	}
	// Also reclaim stuck link previews when Redis dropped the original unfurl task.
	if unfurled, uerr := h.Messages.ProcessPendingLinkPreviews(ctx, 20); uerr != nil {
		h.Logger.Error("link:unfurl reclaim failed", "error", uerr)
		return uerr
	} else if unfurled > 0 {
		h.Logger.Info("reclaimed pending link previews", "count", unfurled)
	}
	return nil
}

func (h *Handlers) handleUnfurlLink(ctx context.Context, task *asynq.Task) error {
	if h.Messages == nil {
		return nil
	}
	var payload UnfurlLinkPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		err = fmt.Errorf("decode unfurl payload: %w", err)
		h.Logger.Error("link:unfurl failed", "error", err)
		return err
	}
	if err := h.Messages.ProcessLinkPreview(ctx, payload.PreviewID); err != nil {
		h.Logger.Error("link:unfurl failed", "error", err, "previewId", payload.PreviewID)
		return err
	}
	h.Logger.Info("link:unfurl completed", "previewId", payload.PreviewID)
	return nil
}

func (h *Handlers) handlePurgeExpired(ctx context.Context, _ *asynq.Task) error {
	if h.Billing == nil {
		return nil
	}
	n, err := h.Billing.PurgeExpiredHistory(ctx, time.Now().UTC(), 200)
	if err != nil {
		h.Logger.Error("message:purge_expired failed", "error", err)
		return err
	}
	if n > 0 {
		h.Logger.Info("purged expired messages", "count", n)
	}
	return nil
}

func (h *Handlers) handleReconcileBilling(ctx context.Context, _ *asynq.Task) error {
	if h.Billing == nil {
		return nil
	}
	if err := h.Billing.Reconcile(ctx, time.Now().UTC()); err != nil {
		h.Logger.Error("billing:reconcile failed", "error", err)
		return err
	}
	return nil
}

// SchedulePeriodicPublish enqueues a repeating publish task every interval.
func SchedulePeriodicPublish(scheduler *asynq.Scheduler, interval time.Duration) error {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	entryID, err := scheduler.Register(
		fmt.Sprintf("@every %s", interval),
		NewPublishScheduledTask(),
	)
	if err != nil {
		return err
	}
	_ = entryID
	return nil
}

// SchedulePeriodicMaintenance registers retention and billing reconcile jobs.
func SchedulePeriodicMaintenance(scheduler *asynq.Scheduler) error {
	if _, err := scheduler.Register("@every 1h", NewPurgeExpiredTask()); err != nil {
		return err
	}
	if _, err := scheduler.Register("@every 15m", NewReconcileBillingTask()); err != nil {
		return err
	}
	return nil
}
