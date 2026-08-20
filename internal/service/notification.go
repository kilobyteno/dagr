package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
)

// NotificationWriter creates inbox notifications for users.
type NotificationWriter interface {
	Notify(ctx context.Context, in NotifyInput) error
}

type NotifyInput struct {
	UserID      string
	ActorID     string
	Kind        domain.NotificationKind
	WorkspaceID string
	ChannelID   string
	MessageID   string
	Body        string
}

type noopNotificationWriter struct{}

func (noopNotificationWriter) Notify(context.Context, NotifyInput) error { return nil }

type NotificationStore interface {
	CreateNotification(ctx context.Context, in postgres.CreateNotificationInput) (postgres.NotificationRow, error)
	ListNotifications(ctx context.Context, userID uuid.UUID, filter string, limit int) ([]postgres.NotificationRow, error)
	CountUnreadNotifications(ctx context.Context, userID uuid.UUID) (int, error)
	MarkNotificationRead(ctx context.Context, userID, notificationID uuid.UUID) error
	MarkAllNotificationsRead(ctx context.Context, userID uuid.UUID) error
	MarkNotificationsReadForChannel(ctx context.Context, userID, channelID uuid.UUID) error
	ListWorkspaceMembers(ctx context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceMemberInfo, error)
	GetChannel(ctx context.Context, channelID uuid.UUID) (postgres.ChannelRow, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (postgres.UserRow, error)
	ListUserNotificationLevels(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]domain.NotificationLevel, error)
	ListChannelMemberIDs(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error)
	GetChannelNotificationLevel(ctx context.Context, userID, channelID uuid.UUID) (domain.ChannelNotificationLevel, error)
	UpsertChannelNotificationLevel(ctx context.Context, userID, channelID uuid.UUID, level domain.ChannelNotificationLevel) error
	ListChannelNotificationLevels(ctx context.Context, channelID uuid.UUID) (map[uuid.UUID]domain.ChannelNotificationLevel, error)
}

type NotificationService struct {
	store NotificationStore
}

func NewNotificationService(store NotificationStore) *NotificationService {
	return &NotificationService{store: store}
}

// Notify implements NotificationWriter.
func (s *NotificationService) Notify(ctx context.Context, in NotifyInput) error {
	uid, err := uuid.Parse(in.UserID)
	if err != nil {
		return ErrInvalidInput
	}
	if globals, err := s.store.ListUserNotificationLevels(ctx, []uuid.UUID{uid}); err == nil {
		if globalLevel(globals, uid) == domain.NotifyNothing {
			return nil
		}
	}
	payload := postgres.CreateNotificationInput{
		UserID: uid,
		Kind:   in.Kind,
		Body:   strings.TrimSpace(in.Body),
	}
	if payload.Body == "" {
		return ErrInvalidInput
	}
	if in.ActorID != "" {
		aid, err := uuid.Parse(in.ActorID)
		if err != nil {
			return ErrInvalidInput
		}
		payload.ActorID = &aid
	}
	if in.WorkspaceID != "" {
		wid, err := uuid.Parse(in.WorkspaceID)
		if err != nil {
			return ErrInvalidInput
		}
		payload.WorkspaceID = &wid
	}
	if in.ChannelID != "" {
		cid, err := uuid.Parse(in.ChannelID)
		if err != nil {
			return ErrInvalidInput
		}
		payload.ChannelID = &cid
	}
	if in.MessageID != "" {
		mid, err := uuid.Parse(in.MessageID)
		if err != nil {
			return ErrInvalidInput
		}
		payload.MessageID = &mid
	}
	_, err = s.store.CreateNotification(ctx, payload)
	return err
}

func (s *NotificationService) List(ctx context.Context, userID, filter string, limit int) ([]domain.Notification, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	switch filter {
	case "", "all", "unread", "mentions":
		if filter == "" {
			filter = "all"
		}
	default:
		return nil, ErrInvalidInput
	}
	rows, err := s.store.ListNotifications(ctx, uid, filter, limit)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Notification, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ToDomain())
	}
	return out, nil
}

func (s *NotificationService) UnreadCount(ctx context.Context, userID string) (int, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return 0, ErrInvalidInput
	}
	return s.store.CountUnreadNotifications(ctx, uid)
}

func (s *NotificationService) MarkRead(ctx context.Context, userID, notificationID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidInput
	}
	nid, err := uuid.Parse(notificationID)
	if err != nil {
		return ErrNotFound
	}
	if err := s.store.MarkNotificationRead(ctx, uid, nid); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidInput
	}
	return s.store.MarkAllNotificationsRead(ctx, uid)
}

func (s *NotificationService) MarkChannelRead(ctx context.Context, userID, channelID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidInput
	}
	cid, err := uuid.Parse(channelID)
	if err != nil {
		return ErrInvalidInput
	}
	return s.store.MarkNotificationsReadForChannel(ctx, uid, cid)
}

func (s *NotificationService) GetChannelNotificationLevel(
	ctx context.Context, userID, channelID string,
) (domain.ChannelNotificationLevel, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", ErrInvalidInput
	}
	cid, err := uuid.Parse(channelID)
	if err != nil {
		return "", ErrInvalidInput
	}
	if _, err := s.store.GetChannel(ctx, cid); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}
	return s.store.GetChannelNotificationLevel(ctx, uid, cid)
}

func (s *NotificationService) SetChannelNotificationLevel(
	ctx context.Context, userID, channelID, level string,
) (domain.ChannelNotificationLevel, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", ErrInvalidInput
	}
	cid, err := uuid.Parse(channelID)
	if err != nil {
		return "", ErrInvalidInput
	}
	parsed, ok := domain.ParseChannelNotificationLevel(level)
	if !ok {
		return "", ErrInvalidInput
	}
	if _, err := s.store.GetChannel(ctx, cid); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}
	if err := s.store.UpsertChannelNotificationLevel(ctx, uid, cid, parsed); err != nil {
		return "", err
	}
	return parsed, nil
}

// EmitMentions resolves @mentions and all-message prefs for a posted message.
// Direct messages notify the peer for every message unless the effective level is "nothing".
func (s *NotificationService) EmitMentions(
	ctx context.Context,
	authorID, authorName, workspaceID, channelID, channelName, messageID, body string,
) error {
	cid := mustUUID(channelID)
	ch, err := s.store.GetChannel(ctx, cid)
	if err != nil {
		return err
	}
	isDM := ch.Kind == "dm"
	members, err := s.store.ListWorkspaceMembers(ctx, mustUUID(workspaceID))
	if err != nil {
		return err
	}
	channelLevels, err := s.store.ListChannelNotificationLevels(ctx, cid)
	if err != nil {
		return err
	}
	eligible := map[uuid.UUID]struct{}{}
	if ch.IsPrivate {
		memberIDs, err := s.store.ListChannelMemberIDs(ctx, cid)
		if err != nil {
			return err
		}
		for _, id := range memberIDs {
			eligible[id] = struct{}{}
		}
	} else {
		for _, m := range members {
			eligible[m.UserID] = struct{}{}
		}
	}

	eligibleIDs := make([]uuid.UUID, 0, len(eligible))
	for id := range eligible {
		eligibleIDs = append(eligibleIDs, id)
	}
	globalLevels, err := s.store.ListUserNotificationLevels(ctx, eligibleIDs)
	if err != nil {
		return err
	}

	mentioned := resolveMentionedUsers(body, members, authorID)
	mentionedIDs := map[uuid.UUID]struct{}{}
	snippet := truncateRunes(body, 120)
	for _, member := range mentioned {
		if _, ok := eligible[member.UserID]; !ok {
			continue
		}
		level := effectiveNotifyLevel(globalLevels, channelLevels, member.UserID)
		if level == domain.NotifyNothing {
			continue
		}
		// Mentions notify for both "mentions" and "all".
		mentionedIDs[member.UserID] = struct{}{}
		notifyBody := mentionNotificationBody(authorName, channelName, snippet, isDM)
		if err := s.Notify(ctx, NotifyInput{
			UserID:      member.UserID.String(),
			ActorID:     authorID,
			Kind:        domain.NotificationMention,
			WorkspaceID: workspaceID,
			ChannelID:   channelID,
			MessageID:   messageID,
			Body:        notifyBody,
		}); err != nil {
			return err
		}
	}

	authorUUID := mustUUID(authorID)
	for userID := range eligible {
		if userID == authorUUID {
			continue
		}
		if _, already := mentionedIDs[userID]; already {
			continue
		}
		level := effectiveNotifyLevel(globalLevels, channelLevels, userID)
		if level == domain.NotifyNothing {
			continue
		}
		// Channels require "all"; DMs notify on every message unless silenced.
		if !isDM && level != domain.NotifyAll {
			continue
		}
		notifyBody := messageNotificationBody(authorName, channelName, snippet, isDM)
		if err := s.Notify(ctx, NotifyInput{
			UserID:      userID.String(),
			ActorID:     authorID,
			Kind:        domain.NotificationMessage,
			WorkspaceID: workspaceID,
			ChannelID:   channelID,
			MessageID:   messageID,
			Body:        notifyBody,
		}); err != nil {
			return err
		}
	}
	return nil
}

func mentionNotificationBody(authorName, channelName, snippet string, isDM bool) string {
	if isDM {
		if authorName != "" {
			return fmt.Sprintf("%s mentioned you: \"%s\"", authorName, snippet)
		}
		return fmt.Sprintf("Mentioned you: \"%s\"", snippet)
	}
	if authorName != "" {
		return fmt.Sprintf("%s mentioned you in #%s: \"%s\"", authorName, channelName, snippet)
	}
	return fmt.Sprintf("Mentioned you in #%s: \"%s\"", channelName, snippet)
}

func messageNotificationBody(authorName, channelName, snippet string, isDM bool) string {
	if isDM {
		if authorName != "" {
			return fmt.Sprintf("%s: \"%s\"", authorName, snippet)
		}
		return fmt.Sprintf("New direct message: \"%s\"", snippet)
	}
	if authorName != "" {
		return fmt.Sprintf("%s in #%s: \"%s\"", authorName, channelName, snippet)
	}
	return fmt.Sprintf("New message in #%s: \"%s\"", channelName, snippet)
}

// EmitReaction notifies the message author when someone reacts (not themselves).
// Honours effective notification level: silenced when global/channel is "nothing".
func (s *NotificationService) EmitReaction(
	ctx context.Context,
	reactorID, reactorName, authorID, workspaceID, channelID, channelName, messageID, emoji string,
) error {
	if reactorID == "" || authorID == "" || reactorID == authorID {
		return nil
	}
	authorUUID, err := uuid.Parse(authorID)
	if err != nil {
		return ErrInvalidInput
	}
	cid, err := uuid.Parse(channelID)
	if err != nil {
		return ErrInvalidInput
	}
	globals, err := s.store.ListUserNotificationLevels(ctx, []uuid.UUID{authorUUID})
	if err != nil {
		return err
	}
	channelLevel, err := s.store.GetChannelNotificationLevel(ctx, authorUUID, cid)
	if err != nil {
		return err
	}
	channelLevels := map[uuid.UUID]domain.ChannelNotificationLevel{
		authorUUID: channelLevel,
	}
	if effectiveNotifyLevel(globals, channelLevels, authorUUID) == domain.NotifyNothing {
		return nil
	}
	emojiLabel := strings.Trim(strings.TrimSpace(emoji), ":")
	if emojiLabel == "" {
		emojiLabel = "emoji"
	}
	ch, err := s.store.GetChannel(ctx, cid)
	if err != nil {
		return err
	}
	isDM := ch.Kind == "dm"
	var notifyBody string
	if isDM {
		notifyBody = fmt.Sprintf("Reacted with :%s: to your message", emojiLabel)
		if reactorName != "" {
			notifyBody = fmt.Sprintf(
				"%s reacted with :%s: to your message",
				reactorName, emojiLabel,
			)
		}
	} else {
		notifyBody = fmt.Sprintf("Reacted with :%s: to your message in #%s", emojiLabel, channelName)
		if reactorName != "" {
			notifyBody = fmt.Sprintf(
				"%s reacted with :%s: to your message in #%s",
				reactorName, emojiLabel, channelName,
			)
		}
	}
	return s.Notify(ctx, NotifyInput{
		UserID:      authorID,
		ActorID:     reactorID,
		Kind:        domain.NotificationReaction,
		WorkspaceID: workspaceID,
		ChannelID:   channelID,
		MessageID:   messageID,
		Body:        notifyBody,
	})
}

func globalLevel(
	levels map[uuid.UUID]domain.NotificationLevel,
	userID uuid.UUID,
) domain.NotificationLevel {
	if level, ok := levels[userID]; ok {
		return level
	}
	return domain.NotifyMentions
}

func channelNotifyLevel(
	levels map[uuid.UUID]domain.ChannelNotificationLevel,
	userID uuid.UUID,
) domain.ChannelNotificationLevel {
	if level, ok := levels[userID]; ok {
		return level
	}
	return domain.ChannelNotifyMentions
}

func effectiveNotifyLevel(
	globals map[uuid.UUID]domain.NotificationLevel,
	channels map[uuid.UUID]domain.ChannelNotificationLevel,
	userID uuid.UUID,
) domain.NotificationLevel {
	return domain.MinNotificationLevel(
		globalLevel(globals, userID),
		channelNotifyLevel(channels, userID),
	)
}

func resolveMentionedUsers(
	body string,
	members []postgres.WorkspaceMemberInfo,
	authorID string,
) []postgres.WorkspaceMemberInfo {
	type named struct {
		lower string
		info  postgres.WorkspaceMemberInfo
	}
	handles := map[string]postgres.WorkspaceMemberInfo{}
	names := make([]named, 0, len(members))
	byFirst := map[string][]postgres.WorkspaceMemberInfo{}
	for _, m := range members {
		handle := strings.ToLower(strings.TrimSpace(m.Handle))
		if handle != "" {
			handles[handle] = m
		}
		for _, former := range m.FormerHandles {
			key := strings.ToLower(strings.TrimSpace(former))
			if key == "" {
				continue
			}
			// Current handles win if both somehow collide.
			if _, exists := handles[key]; !exists {
				handles[key] = m
			}
		}
		key := strings.ToLower(strings.TrimSpace(m.DisplayName))
		if key == "" {
			continue
		}
		names = append(names, named{lower: key, info: m})
		parts := strings.Fields(key)
		if len(parts) > 0 {
			byFirst[parts[0]] = append(byFirst[parts[0]], m)
		}
	}
	// Prefer longer display names so "@Ada Lovelace" wins over "@Ada".
	sort.Slice(names, func(i, j int) bool {
		return len(names[i].lower) > len(names[j].lower)
	})

	lowerBody := strings.ToLower(body)
	seen := map[uuid.UUID]struct{}{}
	var out []postgres.WorkspaceMemberInfo
	add := func(m postgres.WorkspaceMemberInfo) {
		if m.UserID.String() == authorID {
			return
		}
		if _, ok := seen[m.UserID]; ok {
			return
		}
		seen[m.UserID] = struct{}{}
		out = append(out, m)
	}

	for i := 0; i < len(lowerBody); i++ {
		if lowerBody[i] != '@' {
			continue
		}
		rest := lowerBody[i+1:]
		token := ""
		for _, r := range rest {
			if r == ' ' || r == '\n' || r == '\t' || r == ',' || r == '.' || r == '!' || r == '?' || r == ':' || r == ';' {
				break
			}
			token += string(r)
		}
		if token == "" {
			continue
		}
		// Prefer workspace handles: @ada
		if member, ok := handles[token]; ok {
			add(member)
			continue
		}
		matched := false
		for _, n := range names {
			if strings.HasPrefix(rest, n.lower) {
				end := len(n.lower)
				if end < len(rest) {
					next := rest[end]
					if next != ' ' && next != ',' && next != '.' && next != '!' && next != '?' && next != ':' && next != ';' && next != '\n' && next != '\t' {
						continue
					}
				}
				add(n.info)
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		// Unambiguous first-token mention: @Ada
		candidates := byFirst[token]
		if len(candidates) == 1 {
			add(candidates[0])
		}
	}
	return out
}

func truncateRunes(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max-1]) + "…"
}

func mustUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
