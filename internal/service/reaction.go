package service

import (
	"context"
	"errors"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
)

func (s *MessageService) emitReactionNotification(
	ctx context.Context,
	reactorID, emoji string,
	msg postgres.MessageRow,
) {
	if s.mentions == nil {
		return
	}
	if msg.AuthorID.String() == reactorID {
		return
	}
	ch, err := s.store.GetChannel(ctx, msg.ChannelID)
	if err != nil {
		return
	}
	reactorName := ""
	if uid, err := uuid.Parse(reactorID); err == nil {
		if user, err := s.store.GetUserByID(ctx, uid); err == nil {
			reactorName = user.DisplayName
		}
	}
	_ = s.mentions.EmitReaction(
		ctx,
		reactorID,
		reactorName,
		msg.AuthorID.String(),
		ch.WorkspaceID.String(),
		ch.ID.String(),
		ch.Name,
		msg.ID.String(),
		emoji,
	)
}

type reactionStore interface {
	AddMessageReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	RemoveMessageReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	ListReactionsForMessages(ctx context.Context, messageIDs []uuid.UUID) ([]postgres.MessageReactionRow, error)
	HasMessageReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (bool, error)
}

func normaliseReactionEmoji(raw string) (string, error) {
	emoji := strings.TrimSpace(strings.ToLower(raw))
	emoji = strings.Trim(emoji, ":")
	if emoji == "" || len(emoji) > 64 {
		return "", ErrInvalidInput
	}
	for _, r := range emoji {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '+' || r == '-' {
			continue
		}
		return "", ErrInvalidInput
	}
	return emoji, nil
}

func (s *MessageService) AddReaction(
	ctx context.Context,
	userID, messageID, emoji string,
) (*domain.Message, error) {
	emoji, err := normaliseReactionEmoji(emoji)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	mid, err := uuid.Parse(messageID)
	if err != nil {
		return nil, ErrNotFound
	}
	msg, err := s.store.GetMessage(ctx, mid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if msg.ContentType == domain.ContentTypeSystem {
		return nil, ErrForbidden
	}
	if _, err := s.channels.CanAccessChannel(ctx, userID, msg.ChannelID.String()); err != nil {
		return nil, err
	}
	store, ok := s.store.(reactionStore)
	if !ok {
		return nil, errors.New("reactions unsupported")
	}
	exists, err := store.HasMessageReaction(ctx, mid, uid, emoji)
	if err != nil {
		return nil, err
	}
	if err := store.AddMessageReaction(ctx, mid, uid, emoji); err != nil {
		return nil, err
	}
	if !exists {
		s.emitReactionNotification(ctx, userID, emoji, msg)
	}
	return s.messageWithAttachments(ctx, userID, msg)
}

func (s *MessageService) RemoveReaction(
	ctx context.Context,
	userID, messageID, emoji string,
) (*domain.Message, error) {
	emoji, err := normaliseReactionEmoji(emoji)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	mid, err := uuid.Parse(messageID)
	if err != nil {
		return nil, ErrNotFound
	}
	msg, err := s.store.GetMessage(ctx, mid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if _, err := s.channels.CanAccessChannel(ctx, userID, msg.ChannelID.String()); err != nil {
		return nil, err
	}
	store, ok := s.store.(reactionStore)
	if !ok {
		return nil, errors.New("reactions unsupported")
	}
	if err := store.RemoveMessageReaction(ctx, mid, uid, emoji); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.messageWithAttachments(ctx, userID, msg)
}

func (s *MessageService) ToggleReaction(
	ctx context.Context,
	userID, messageID, emoji string,
) (*domain.Message, error) {
	emoji, err := normaliseReactionEmoji(emoji)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	mid, err := uuid.Parse(messageID)
	if err != nil {
		return nil, ErrNotFound
	}
	msg, err := s.store.GetMessage(ctx, mid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if msg.ContentType == domain.ContentTypeSystem {
		return nil, ErrForbidden
	}
	if _, err := s.channels.CanAccessChannel(ctx, userID, msg.ChannelID.String()); err != nil {
		return nil, err
	}
	store, ok := s.store.(reactionStore)
	if !ok {
		return nil, errors.New("reactions unsupported")
	}
	exists, err := store.HasMessageReaction(ctx, mid, uid, emoji)
	if err != nil {
		return nil, err
	}
	if exists {
		if err := store.RemoveMessageReaction(ctx, mid, uid, emoji); err != nil && !errors.Is(err, postgres.ErrNotFound) {
			return nil, err
		}
	} else {
		if err := store.AddMessageReaction(ctx, mid, uid, emoji); err != nil {
			return nil, err
		}
		s.emitReactionNotification(ctx, userID, emoji, msg)
	}
	return s.messageWithAttachments(ctx, userID, msg)
}

func (s *MessageService) messageWithAttachments(
	ctx context.Context, viewerID string, row postgres.MessageRow,
) (*domain.Message, error) {
	msg := row.ToDomain()
	msgs := s.attachLinkPreviews(ctx, []domain.Message{msg})
	msgs = s.attachReactions(ctx, viewerID, msgs)
	if len(msgs) == 1 {
		msg = msgs[0]
	}
	return &msg, nil
}

func (s *MessageService) attachReactions(
	ctx context.Context, viewerID string, messages []domain.Message,
) []domain.Message {
	if len(messages) == 0 {
		return messages
	}
	store, ok := s.store.(reactionStore)
	if !ok {
		return messages
	}
	ids := make([]uuid.UUID, 0, len(messages))
	index := map[uuid.UUID]int{}
	for i, msg := range messages {
		id, err := uuid.Parse(msg.ID)
		if err != nil {
			continue
		}
		ids = append(ids, id)
		index[id] = i
		messages[i].Reactions = nil
	}
	rows, err := store.ListReactionsForMessages(ctx, ids)
	if err != nil {
		return messages
	}

	type key struct {
		messageID uuid.UUID
		emoji     string
	}
	order := make([]key, 0)
	grouped := map[key]*domain.MessageReaction{}
	for _, row := range rows {
		k := key{messageID: row.MessageID, emoji: row.Emoji}
		agg, ok := grouped[k]
		if !ok {
			agg = &domain.MessageReaction{
				Emoji:   row.Emoji,
				UserIDs: make([]string, 0, 4),
			}
			grouped[k] = agg
			order = append(order, k)
		}
		agg.Count++
		agg.UserIDs = append(agg.UserIDs, row.UserID.String())
		if row.UserID.String() == viewerID {
			agg.Reacted = true
		}
	}
	for _, k := range order {
		i, ok := index[k.messageID]
		if !ok {
			continue
		}
		messages[i].Reactions = append(messages[i].Reactions, *grouped[k])
	}
	return messages
}
