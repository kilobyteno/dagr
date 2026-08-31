package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
)

const maxScheduleHorizon = 365 * 24 * time.Hour

type MessageStore interface {
	InsertMessage(ctx context.Context, channelID, authorID uuid.UUID, body, contentType string) (postgres.MessageRow, error)
	InsertAppMessage(ctx context.Context, channelID, authorID uuid.UUID, body, contentType string, payload []byte) (postgres.MessageRow, error)
	ListMessages(ctx context.Context, channelID uuid.UUID, before *time.Time, beforeID *uuid.UUID, after *time.Time, limit int) ([]postgres.MessageRow, error)
	GetMessage(ctx context.Context, messageID uuid.UUID) (postgres.MessageRow, error)
	UpdateMessageBody(ctx context.Context, messageID uuid.UUID, body string) (postgres.MessageRow, error)
	DeleteMessage(ctx context.Context, messageID uuid.UUID) error
	InsertScheduledMessage(ctx context.Context, channelID, authorID uuid.UUID, body, contentType string, sendAt time.Time) (postgres.ScheduledMessageRow, error)
	ListScheduledMessages(ctx context.Context, channelID uuid.UUID) ([]postgres.ScheduledMessageRow, error)
	GetScheduledMessage(ctx context.Context, id uuid.UUID) (postgres.ScheduledMessageRow, error)
	CancelScheduledMessage(ctx context.Context, id uuid.UUID) error
	ClaimAndPublishDueScheduledMessages(ctx context.Context, now time.Time, limit int) ([]postgres.PublishedScheduledMessage, error)
	GetChannel(ctx context.Context, channelID uuid.UUID) (postgres.ChannelRow, error)
	IsWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, string, error)
	IsChannelMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (postgres.UserRow, error)
	UpsertChannelReadState(ctx context.Context, userID, channelID uuid.UUID, lastReadMessageID *uuid.UUID) error
	GetChannelReadState(ctx context.Context, userID, channelID uuid.UUID) (postgres.ChannelReadState, error)
	GetPreviousMessageID(ctx context.Context, channelID, messageID uuid.UUID) (*uuid.UUID, error)
	GetLatestMessageID(ctx context.Context, channelID uuid.UUID) (*uuid.UUID, error)
	GetChannelUnreadSummary(ctx context.Context, userID, channelID uuid.UUID) (int, *uuid.UUID, error)
}

type EntitlementLookup interface {
	ForWorkspace(ctx context.Context, workspaceID string) domain.Entitlements
}

type MessageListResult struct {
	Messages             []domain.Message
	HistoryLimited       bool
	HistoryRetentionDays *int
}

type MessageService struct {
	store        MessageStore
	channels     *ChannelService
	notify       NotificationWriter
	mentions     *NotificationService
	unfurl       LinkUnfurlEnqueuer
	entitlements EntitlementLookup
}

func NewMessageService(store MessageStore, channels *ChannelService) *MessageService {
	return &MessageService{store: store, channels: channels, notify: noopNotificationWriter{}}
}

func (s *MessageService) WithEntitlements(lookup EntitlementLookup) *MessageService {
	s.entitlements = lookup
	return s
}

func (s *MessageService) WithNotifications(notify NotificationWriter, mentions *NotificationService) *MessageService {
	if notify != nil {
		s.notify = notify
	}
	s.mentions = mentions
	return s
}

func (s *MessageService) Post(ctx context.Context, userID, channelID, body string) (*domain.Message, error) {
	body = strings.TrimSpace(body)
	if body == "" || len(body) > 8000 {
		return nil, ErrInvalidInput
	}
	ch, err := s.channels.CanAccessChannel(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	uid, _ := uuid.Parse(userID)
	cid, _ := uuid.Parse(channelID)
	row, err := s.store.InsertMessage(ctx, cid, uid, body, "text/plain")
	if err != nil {
		return nil, err
	}
	s.emitMentionsForMessage(ctx, uid, ch, row)
	s.queueLinkPreviews(ctx, row.ID, row.Body, row.ContentType)
	return s.messageWithAttachments(ctx, userID, row)
}

func (s *MessageService) PostFromApp(ctx context.Context, authorID, channelID string, payload domain.RichPayload) (*domain.Message, error) {
	uid, err := uuid.Parse(authorID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	cid, err := uuid.Parse(channelID)
	if err != nil {
		return nil, ErrNotFound
	}
	chRow, err := s.store.GetChannel(ctx, cid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if chRow.Kind == "dm" {
		return nil, ErrNotAChannel
	}
	body := FallbackBody(payload)
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	row, err := s.store.InsertAppMessage(ctx, cid, uid, body, domain.ContentTypeRich, raw)
	if err != nil {
		return nil, err
	}
	ch := chRow.ToDomain()
	s.emitMentionsForMessage(ctx, uid, &ch, row)
	return s.messageWithAttachments(ctx, authorID, row)
}

func (s *MessageService) emitMentionsForMessage(
	ctx context.Context, authorID uuid.UUID, ch *domain.Channel, row postgres.MessageRow,
) {
	if s.mentions == nil || ch == nil {
		return
	}
	if row.ContentType == domain.ContentTypeSystem {
		return
	}
	authorName := row.AuthorName
	if authorName == "" {
		if user, err := s.store.GetUserByID(ctx, authorID); err == nil {
			authorName = user.DisplayName
		}
	}
	_ = s.mentions.EmitMentions(
		ctx,
		authorID.String(),
		authorName,
		ch.WorkspaceID,
		ch.ID,
		ch.Name,
		row.ID.String(),
		row.Body,
	)
}

func (s *MessageService) List(
	ctx context.Context,
	userID, channelID string,
	before *time.Time,
	beforeID *uuid.UUID,
	limit int,
) ([]domain.Message, error) {
	result, err := s.ListWithMeta(ctx, userID, channelID, before, beforeID, limit)
	if err != nil {
		return nil, err
	}
	return result.Messages, nil
}

func (s *MessageService) ListWithMeta(
	ctx context.Context,
	userID, channelID string,
	before *time.Time,
	beforeID *uuid.UUID,
	limit int,
) (MessageListResult, error) {
	if _, err := s.channels.CanAccessChannel(ctx, userID, channelID); err != nil {
		return MessageListResult{}, err
	}
	cid, _ := uuid.Parse(channelID)
	after, days := s.historyFloor(ctx, cid)
	rows, err := s.store.ListMessages(ctx, cid, before, beforeID, after, limit)
	if err != nil {
		return MessageListResult{}, err
	}
	out := make([]domain.Message, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		out = append(out, rows[i].ToDomain())
	}
	out = s.attachLinkPreviews(ctx, out)
	return MessageListResult{
		Messages:             s.attachReactions(ctx, userID, out),
		HistoryLimited:       days != nil,
		HistoryRetentionDays: days,
	}, nil
}

func (s *MessageService) historyFloor(ctx context.Context, channelID uuid.UUID) (*time.Time, *int) {
	if s.entitlements == nil {
		return nil, nil
	}
	ch, err := s.store.GetChannel(ctx, channelID)
	if err != nil {
		return nil, nil
	}
	ents := s.entitlements.ForWorkspace(ctx, ch.WorkspaceID.String())
	if ents.UnlimitedHistory || ents.MessageHistoryDays == nil {
		return nil, nil
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -*ents.MessageHistoryDays)
	return &cutoff, ents.MessageHistoryDays
}

func (s *MessageService) Update(
	ctx context.Context,
	userID, messageID, body string,
) (*domain.Message, error) {
	body = strings.TrimSpace(body)
	if body == "" || len(body) > 8000 {
		return nil, ErrInvalidInput
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	mid, err := uuid.Parse(messageID)
	if err != nil {
		return nil, ErrNotFound
	}
	current, err := s.store.GetMessage(ctx, mid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if _, err := s.channels.CanAccessChannel(ctx, userID, current.ChannelID.String()); err != nil {
		return nil, err
	}
	if current.AuthorID != uid {
		return nil, ErrForbidden
	}
	if current.ContentType == domain.ContentTypeSystem || current.ContentType == domain.ContentTypeRich {
		return nil, ErrForbidden
	}
	row, err := s.store.UpdateMessageBody(ctx, mid, body)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	s.queueLinkPreviews(ctx, row.ID, row.Body, row.ContentType)
	return s.messageWithAttachments(ctx, userID, row)
}

func (s *MessageService) Delete(ctx context.Context, userID, messageID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidInput
	}
	mid, err := uuid.Parse(messageID)
	if err != nil {
		return ErrNotFound
	}
	current, err := s.store.GetMessage(ctx, mid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if _, err := s.channels.CanAccessChannel(ctx, userID, current.ChannelID.String()); err != nil {
		return err
	}
	if current.AuthorID != uid {
		return ErrForbidden
	}
	if current.ContentType == domain.ContentTypeSystem {
		return ErrForbidden
	}
	if err := s.store.DeleteMessage(ctx, mid); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// ChannelUnread is the per-user unread cursor summary for a channel.
type ChannelUnread struct {
	UnreadCount          int
	FirstUnreadMessageID string
}

func (s *MessageService) MarkRead(
	ctx context.Context,
	userID, channelID, messageID string,
) (*ChannelUnread, error) {
	if _, err := s.channels.CanAccessChannel(ctx, userID, channelID); err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	cid, err := uuid.Parse(channelID)
	if err != nil {
		return nil, ErrNotFound
	}

	var lastRead *uuid.UUID
	if strings.TrimSpace(messageID) == "" {
		latest, err := s.store.GetLatestMessageID(ctx, cid)
		if err != nil {
			return nil, err
		}
		lastRead = latest
	} else {
		mid, err := uuid.Parse(messageID)
		if err != nil {
			return nil, ErrInvalidInput
		}
		msg, err := s.store.GetMessage(ctx, mid)
		if err != nil {
			if errors.Is(err, postgres.ErrNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		if msg.ChannelID != cid {
			return nil, ErrNotFound
		}
		current, err := s.store.GetChannelReadState(ctx, uid, cid)
		if err != nil {
			return nil, err
		}
		lastRead = &mid
		if current.HasRow && current.LastReadMessageID != nil {
			existing, err := s.store.GetMessage(ctx, *current.LastReadMessageID)
			if err == nil && postgres.MessageIsAtOrBefore(msg, existing) {
				lastRead = current.LastReadMessageID
			}
		}
	}

	if err := s.store.UpsertChannelReadState(ctx, uid, cid, lastRead); err != nil {
		return nil, err
	}
	if s.mentions != nil {
		_ = s.mentions.MarkChannelRead(ctx, userID, channelID)
	}
	return s.unreadSummary(ctx, uid, cid)
}

func (s *MessageService) MarkUnread(
	ctx context.Context,
	userID, channelID, messageID string,
) (*ChannelUnread, error) {
	if _, err := s.channels.CanAccessChannel(ctx, userID, channelID); err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	cid, err := uuid.Parse(channelID)
	if err != nil {
		return nil, ErrNotFound
	}
	mid, err := uuid.Parse(messageID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	msg, err := s.store.GetMessage(ctx, mid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if msg.ChannelID != cid {
		return nil, ErrNotFound
	}
	if msg.AuthorID == uid {
		return nil, ErrForbidden
	}
	prev, err := s.store.GetPreviousMessageID(ctx, cid, mid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := s.store.UpsertChannelReadState(ctx, uid, cid, prev); err != nil {
		return nil, err
	}
	return s.unreadSummary(ctx, uid, cid)
}

func (s *MessageService) unreadSummary(
	ctx context.Context, userID, channelID uuid.UUID,
) (*ChannelUnread, error) {
	count, first, err := s.store.GetChannelUnreadSummary(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	out := &ChannelUnread{UnreadCount: count}
	if first != nil {
		out.FirstUnreadMessageID = first.String()
	}
	return out, nil
}

func (s *MessageService) Schedule(
	ctx context.Context,
	userID, channelID, body string,
	sendAt time.Time,
) (*domain.ScheduledMessage, error) {
	body = strings.TrimSpace(body)
	if body == "" || len(body) > 8000 {
		return nil, ErrInvalidInput
	}
	now := time.Now().UTC()
	if !sendAt.After(now.Add(30*time.Second)) || sendAt.After(now.Add(maxScheduleHorizon)) {
		return nil, ErrInvalidInput
	}
	if _, err := s.channels.CanAccessChannel(ctx, userID, channelID); err != nil {
		return nil, err
	}
	uid, _ := uuid.Parse(userID)
	cid, _ := uuid.Parse(channelID)
	row, err := s.store.InsertScheduledMessage(ctx, cid, uid, body, "text/plain", sendAt.UTC())
	if err != nil {
		return nil, err
	}
	out := row.ToDomain()
	return &out, nil
}

func (s *MessageService) ListScheduled(ctx context.Context, userID, channelID string) ([]domain.ScheduledMessage, error) {
	if _, err := s.channels.CanAccessChannel(ctx, userID, channelID); err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	cid, _ := uuid.Parse(channelID)
	ch, err := s.store.GetChannel(ctx, cid)
	if err != nil {
		return nil, err
	}
	ok, role, err := s.store.IsWorkspaceMember(ctx, ch.WorkspaceID, uid)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	isAdmin := canManageWorkspace(domain.WorkspaceRole(role))
	rows, err := s.store.ListScheduledMessages(ctx, cid)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ScheduledMessage, 0, len(rows))
	for _, row := range rows {
		if !isAdmin && row.AuthorID != uid {
			continue
		}
		out = append(out, row.ToDomain())
	}
	return out, nil
}

func (s *MessageService) CancelScheduled(ctx context.Context, userID, scheduledID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidInput
	}
	sid, err := uuid.Parse(scheduledID)
	if err != nil {
		return ErrNotFound
	}
	row, err := s.store.GetScheduledMessage(ctx, sid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if _, err := s.channels.CanAccessChannel(ctx, userID, row.ChannelID.String()); err != nil {
		return err
	}
	ch, err := s.store.GetChannel(ctx, row.ChannelID)
	if err != nil {
		return err
	}
	ok, role, err := s.store.IsWorkspaceMember(ctx, ch.WorkspaceID, uid)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	if row.AuthorID != uid && !canManageWorkspace(domain.WorkspaceRole(role)) {
		return ErrForbidden
	}
	if err := s.store.CancelScheduledMessage(ctx, sid); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// PublishDue is callable by the worker (and tests) to flush due scheduled messages.
func (s *MessageService) PublishDue(ctx context.Context, now time.Time, limit int) (int, error) {
	published, err := s.store.ClaimAndPublishDueScheduledMessages(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	for _, msg := range published {
		chRow, err := s.store.GetChannel(ctx, msg.ChannelID)
		if err != nil {
			continue
		}
		ch := chRow.ToDomain()
		row := postgres.MessageRow{
			ID: msg.ID, ChannelID: msg.ChannelID, AuthorID: msg.AuthorID, Body: msg.Body,
			ContentType: "text/plain",
		}
		s.emitMentionsForMessage(ctx, msg.AuthorID, &ch, row)
		s.queueLinkPreviews(ctx, msg.ID, msg.Body, row.ContentType)
	}
	return len(published), nil
}
