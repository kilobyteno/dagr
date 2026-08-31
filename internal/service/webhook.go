package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/config"
	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
)

var (
	ErrWebhookRateLimited = errors.New("webhook rate limited")
	ErrNotAChannel        = errors.New("not a channel")
)

type WebhookStore interface {
	AppStore
	GetChannel(ctx context.Context, channelID uuid.UUID) (postgres.ChannelRow, error)
	IsChannelMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error)
	IsWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, string, error)
	GetChannelAppInstallForApp(ctx context.Context, workspaceID, channelID, appID uuid.UUID) (postgres.ChannelAppInstallRow, error)
	InsertChannelAppInstall(ctx context.Context, workspaceInstallID, channelID uuid.UUID) (postgres.ChannelAppInstallRow, error)
	DeleteChannelAppInstall(ctx context.Context, installID uuid.UUID) error
	InsertIncomingWebhook(ctx context.Context, channelInstallID uuid.UUID, tokenHash, tokenPrefix string) (postgres.IncomingWebhookRow, error)
	GetIncomingWebhookByChannelInstall(ctx context.Context, channelInstallID uuid.UUID) (postgres.IncomingWebhookRow, error)
	GetIncomingWebhookByTokenHash(ctx context.Context, tokenHash string) (postgres.IncomingWebhookRow, error)
	GetIncomingWebhookBot(ctx context.Context, webhookID uuid.UUID) (uuid.UUID, uuid.UUID, error)
	RotateIncomingWebhook(ctx context.Context, webhookID uuid.UUID, tokenHash, tokenPrefix string) (postgres.IncomingWebhookRow, error)
	TouchIncomingWebhook(ctx context.Context, webhookID uuid.UUID) error
}

type WebhookService struct {
	store    WebhookStore
	apps     *AppService
	messages *MessageService
	cfg      config.Config
	limiter  *webhookLimiter
}

func NewWebhookService(store WebhookStore, apps *AppService, messages *MessageService, cfg config.Config) *WebhookService {
	return &WebhookService{
		store: store, apps: apps, messages: messages, cfg: cfg,
		limiter: newWebhookLimiter(),
	}
}

type IncomingWebhookSecret struct {
	domain.IncomingWebhook
	Token string
}

func (s *WebhookService) GetForChannel(ctx context.Context, userID, channelID string) (*domain.IncomingWebhook, error) {
	_, _, hook, err := s.loadManagedHook(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	out := hook.ToDomain()
	return &out, nil
}

func (s *WebhookService) EnableOnChannel(ctx context.Context, userID, channelID string) (*IncomingWebhookSecret, error) {
	uid, ch, err := s.requireChannelManage(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	install, app, err := s.apps.EnsureInstalled(ctx, userID, ch.WorkspaceID.String(), domain.AppSlugIncomingWebhooks)
	if err != nil {
		return nil, err
	}
	if existing, err := s.store.GetChannelAppInstallForApp(ctx, ch.WorkspaceID, ch.ID, app.ID); err == nil {
		hook, err := s.store.GetIncomingWebhookByChannelInstall(ctx, existing.ID)
		if err != nil {
			return nil, err
		}
		out := hook.ToDomain()
		return &IncomingWebhookSecret{IncomingWebhook: out}, nil
	} else if !errors.Is(err, postgres.ErrNotFound) {
		return nil, err
	}
	channelInstall, err := s.store.InsertChannelAppInstall(ctx, install.ID, ch.ID)
	if err != nil {
		if errors.Is(err, postgres.ErrConflict) {
			return nil, ErrAlreadyInstalled
		}
		return nil, err
	}
	token, prefix, hash, err := newWebhookToken()
	if err != nil {
		return nil, err
	}
	hook, err := s.store.InsertIncomingWebhook(ctx, channelInstall.ID, hash, prefix)
	if err != nil {
		return nil, err
	}
	_ = uid
	out := hook.ToDomain()
	out.URL = s.webhookURL(token)
	return &IncomingWebhookSecret{IncomingWebhook: out, Token: token}, nil
}

func (s *WebhookService) RotateOnChannel(ctx context.Context, userID, channelID string) (*IncomingWebhookSecret, error) {
	_, _, hook, err := s.loadManagedHook(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	token, prefix, hash, err := newWebhookToken()
	if err != nil {
		return nil, err
	}
	rotated, err := s.store.RotateIncomingWebhook(ctx, hook.ID, hash, prefix)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	out := rotated.ToDomain()
	out.URL = s.webhookURL(token)
	return &IncomingWebhookSecret{IncomingWebhook: out, Token: token}, nil
}

func (s *WebhookService) DisableOnChannel(ctx context.Context, userID, channelID string) error {
	_, _, hook, err := s.loadManagedHook(ctx, userID, channelID)
	if err != nil {
		return err
	}
	if err := s.store.DeleteChannelAppInstall(ctx, hook.ChannelAppInstallID); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *WebhookService) Receive(ctx context.Context, token string, raw []byte) (*domain.Message, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrNotFound
	}
	if !s.limiter.allow(token, domain.IncomingWebhookRatePerMinute, time.Minute) {
		return nil, ErrWebhookRateLimited
	}
	hook, err := s.store.GetIncomingWebhookByTokenHash(ctx, hashWebhookToken(token))
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	ch, err := s.store.GetChannel(ctx, hook.ChannelID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if ch.Kind == "dm" {
		return nil, ErrNotAChannel
	}
	payload, err := ParseIncomingWebhookPayload(raw)
	if err != nil {
		return nil, err
	}
	botID, _, err := s.store.GetIncomingWebhookBot(ctx, hook.ID)
	if err != nil {
		return nil, err
	}
	msg, err := s.messages.PostFromApp(ctx, botID.String(), hook.ChannelID.String(), payload)
	if err != nil {
		return nil, err
	}
	_ = s.store.TouchIncomingWebhook(ctx, hook.ID)
	return msg, nil
}

func (s *WebhookService) loadManagedHook(
	ctx context.Context, userID, channelID string,
) (uuid.UUID, postgres.ChannelRow, postgres.IncomingWebhookRow, error) {
	uid, ch, err := s.requireChannelManage(ctx, userID, channelID)
	if err != nil {
		return uuid.Nil, postgres.ChannelRow{}, postgres.IncomingWebhookRow{}, err
	}
	app, err := s.store.GetAppBySlug(ctx, domain.AppSlugIncomingWebhooks)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return uuid.Nil, postgres.ChannelRow{}, postgres.IncomingWebhookRow{}, ErrNotFound
		}
		return uuid.Nil, postgres.ChannelRow{}, postgres.IncomingWebhookRow{}, err
	}
	channelInstall, err := s.store.GetChannelAppInstallForApp(ctx, ch.WorkspaceID, ch.ID, app.ID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return uuid.Nil, postgres.ChannelRow{}, postgres.IncomingWebhookRow{}, ErrNotFound
		}
		return uuid.Nil, postgres.ChannelRow{}, postgres.IncomingWebhookRow{}, err
	}
	hook, err := s.store.GetIncomingWebhookByChannelInstall(ctx, channelInstall.ID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return uuid.Nil, postgres.ChannelRow{}, postgres.IncomingWebhookRow{}, ErrNotFound
		}
		return uuid.Nil, postgres.ChannelRow{}, postgres.IncomingWebhookRow{}, err
	}
	return uid, ch, hook, nil
}

func (s *WebhookService) requireChannelManage(
	ctx context.Context, userID, channelID string,
) (uuid.UUID, postgres.ChannelRow, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, postgres.ChannelRow{}, ErrInvalidInput
	}
	cid, err := uuid.Parse(channelID)
	if err != nil {
		return uuid.Nil, postgres.ChannelRow{}, ErrNotFound
	}
	ch, err := s.store.GetChannel(ctx, cid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return uuid.Nil, postgres.ChannelRow{}, ErrNotFound
		}
		return uuid.Nil, postgres.ChannelRow{}, err
	}
	if ch.Kind == "dm" {
		return uuid.Nil, postgres.ChannelRow{}, ErrNotAChannel
	}
	ok, role, err := s.store.IsWorkspaceMember(ctx, ch.WorkspaceID, uid)
	if err != nil {
		return uuid.Nil, postgres.ChannelRow{}, err
	}
	if !ok {
		return uuid.Nil, postgres.ChannelRow{}, ErrNotFound
	}
	if !canManageWorkspace(domain.WorkspaceRole(role)) {
		return uuid.Nil, postgres.ChannelRow{}, ErrForbidden
	}
	if ch.IsPrivate {
		member, err := s.store.IsChannelMember(ctx, ch.ID, uid)
		if err != nil {
			return uuid.Nil, postgres.ChannelRow{}, err
		}
		if !member {
			return uuid.Nil, postgres.ChannelRow{}, ErrNotFound
		}
	}
	return uid, ch, nil
}

func (s *WebhookService) webhookURL(token string) string {
	base := strings.TrimRight(s.cfg.ServerPublicURL, "/")
	if base == "" {
		base = strings.TrimRight(s.cfg.PublicBaseURL, "/")
	}
	return base + "/api/v1/hooks/" + token
}

func newWebhookToken() (token, prefix, hash string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", "", err
	}
	token = hex.EncodeToString(raw)
	prefix = token[:8]
	hash = hashWebhookToken(token)
	return token, prefix, hash, nil
}

func hashWebhookToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type webhookLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newWebhookLimiter() *webhookLimiter {
	return &webhookLimiter{hits: map[string][]time.Time{}}
}

func (l *webhookLimiter) allow(key string, limit int, window time.Duration) bool {
	now := time.Now()
	cutoff := now.Add(-window)
	l.mu.Lock()
	defer l.mu.Unlock()
	kept := l.hits[key][:0]
	for _, hit := range l.hits[key] {
		if hit.After(cutoff) {
			kept = append(kept, hit)
		}
	}
	if len(kept) >= limit {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}
