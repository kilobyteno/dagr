package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
)

var (
	ErrChannelName = errors.New("invalid channel name")
)

var channelNamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9_-]{0,78}[a-z0-9])?$`)

type ChannelStore interface {
	GetWorkspaceForUser(ctx context.Context, workspaceID, userID uuid.UUID) (postgres.WorkspaceRow, error)
	IsWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, string, error)
	CreateChannel(ctx context.Context, workspaceID, createdBy uuid.UUID, name, topic string, isPrivate bool) (postgres.ChannelRow, error)
	GetChannel(ctx context.Context, channelID uuid.UUID) (postgres.ChannelRow, error)
	GetDMChannelForUser(ctx context.Context, channelID, viewerID uuid.UUID) (postgres.ChannelRow, error)
	FindDMChannel(ctx context.Context, workspaceID, userA, userB uuid.UUID) (postgres.ChannelRow, error)
	CreateDMChannel(ctx context.Context, workspaceID, createdBy, peerID uuid.UUID) (postgres.ChannelRow, error)
	UpdateChannel(ctx context.Context, channelID, updatedBy uuid.UUID, name, topic string, isPrivate bool) (postgres.ChannelRow, error)
	DeleteChannel(ctx context.Context, channelID uuid.UUID) error
	ListChannelsForWorkspace(ctx context.Context, workspaceID, userID uuid.UUID) ([]postgres.ChannelRow, error)
	IsChannelMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error)
	ListChannelMemberIDs(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error)
	ListWorkspaceMembers(ctx context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceMemberInfo, error)
	AddChannelMember(ctx context.Context, channelID, userID uuid.UUID, role domain.ChannelMemberRole) error
	RemoveChannelMember(ctx context.Context, channelID, userID uuid.UUID) error
	GetUserByEmail(ctx context.Context, email string) (postgres.UserRow, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (postgres.UserRow, error)
	AddWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, role domain.WorkspaceRole) error
	InsertMessage(ctx context.Context, channelID, authorID uuid.UUID, body, contentType string) (postgres.MessageRow, error)
}

type ChannelService struct {
	store  ChannelStore
	notify NotificationWriter
}

func NewChannelService(store ChannelStore) *ChannelService {
	return &ChannelService{store: store, notify: noopNotificationWriter{}}
}

func (s *ChannelService) WithNotifications(notify NotificationWriter) *ChannelService {
	if notify != nil {
		s.notify = notify
	}
	return s
}

func (s *ChannelService) Create(
	ctx context.Context,
	userID, workspaceID, name, topic string,
	isPrivate bool,
) (*domain.Channel, error) {
	name, err := normaliseChannelName(name)
	if err != nil {
		return nil, err
	}
	topic = strings.TrimSpace(topic)
	if len(topic) > 250 {
		return nil, ErrInvalidInput
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	ok, _, err := s.store.IsWorkspaceMember(ctx, wid, uid)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	row, err := s.store.CreateChannel(ctx, wid, uid, name, topic, isPrivate)
	if err != nil {
		if errors.Is(err, postgres.ErrChannelNameConflict) {
			return nil, ErrChannelName
		}
		return nil, err
	}
	ch := row.ToDomain()
	return &ch, nil
}

// OpenDM finds or creates a 1:1 DM channel with peerUserID in the workspace.
func (s *ChannelService) OpenDM(
	ctx context.Context, userID, workspaceID, peerUserID string,
) (*domain.Channel, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	peerID, err := uuid.Parse(peerUserID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	if uid == peerID {
		return nil, ErrInvalidInput
	}
	ok, _, err := s.store.IsWorkspaceMember(ctx, wid, uid)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	peerOK, _, err := s.store.IsWorkspaceMember(ctx, wid, peerID)
	if err != nil {
		return nil, err
	}
	if !peerOK {
		return nil, ErrNotFound
	}
	row, err := s.store.FindDMChannel(ctx, wid, uid, peerID)
	if err == nil {
		out := row.ToDomain()
		return &out, nil
	}
	if !errors.Is(err, postgres.ErrNotFound) {
		return nil, err
	}
	row, err = s.store.CreateDMChannel(ctx, wid, uid, peerID)
	if err != nil {
		if errors.Is(err, postgres.ErrChannelNameConflict) {
			existing, findErr := s.store.FindDMChannel(ctx, wid, uid, peerID)
			if findErr != nil {
				return nil, err
			}
			out := existing.ToDomain()
			return &out, nil
		}
		return nil, err
	}
	out := row.ToDomain()
	return &out, nil
}

func (s *ChannelService) Update(
	ctx context.Context,
	userID, channelID, name, topic string,
	isPrivate bool,
) (*domain.Channel, error) {
	ch, uid, err := s.requireChannelManage(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	if ch.Kind == "dm" {
		return nil, ErrInvalidInput
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = ch.Name
	} else {
		name, err = normaliseChannelName(name)
		if err != nil {
			return nil, err
		}
	}
	topic = strings.TrimSpace(topic)
	if len(topic) > 250 {
		return nil, ErrInvalidInput
	}
	before := ch.ToDomain()
	row, err := s.store.UpdateChannel(ctx, ch.ID, uid, name, topic, isPrivate)
	if err != nil {
		if errors.Is(err, postgres.ErrChannelNameConflict) {
			return nil, ErrChannelName
		}
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	out := row.ToDomain()
	s.emitChannelSettingsEvent(ctx, row.ID, uid, before, out)
	return &out, nil
}

func (s *ChannelService) emitChannelSettingsEvent(
	ctx context.Context,
	channelID, actorID uuid.UUID,
	before, after domain.Channel,
) {
	actorName := "Someone"
	if actor, err := s.store.GetUserByID(ctx, actorID); err == nil && actor.DisplayName != "" {
		actorName = actor.DisplayName
	}
	body := channelSettingsEventBody(actorName, before, after)
	if body == "" {
		return
	}
	_, _ = s.store.InsertMessage(ctx, channelID, actorID, body, domain.ContentTypeSystem)
}

func channelSettingsEventBody(actorName string, before, after domain.Channel) string {
	var parts []string
	if before.Name != after.Name {
		parts = append(parts, fmt.Sprintf("renamed the channel to #%s", after.Name))
	}
	if before.Topic != after.Topic {
		if after.Topic == "" {
			parts = append(parts, "cleared the channel topic")
		} else {
			parts = append(parts, fmt.Sprintf("set the channel topic to \"%s\"", after.Topic))
		}
	}
	if before.IsPrivate != after.IsPrivate {
		if after.IsPrivate {
			parts = append(parts, "made this channel private")
		} else {
			parts = append(parts, "made this channel public")
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return actorName + " " + joinNaturalList(parts)
}

func joinNaturalList(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
	}
}

func (s *ChannelService) Delete(ctx context.Context, userID, channelID string) error {
	ch, _, err := s.requireChannelManage(ctx, userID, channelID)
	if err != nil {
		return err
	}
	if ch.Kind == "dm" {
		return ErrInvalidInput
	}
	if err := s.store.DeleteChannel(ctx, ch.ID); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *ChannelService) AddMember(ctx context.Context, actorID, channelID, email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return ErrInvalidInput
	}
	ch, actorUID, err := s.requirePrivateChannelInvite(ctx, actorID, channelID)
	if err != nil {
		return err
	}
	if ch.Kind == "dm" {
		return ErrInvalidInput
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	ok, _, err := s.store.IsWorkspaceMember(ctx, ch.WorkspaceID, user.ID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	if err := s.store.AddChannelMember(ctx, ch.ID, user.ID, domain.ChannelMemberRoleMember); err != nil {
		return err
	}
	actorName := "Someone"
	if actor, err := s.store.GetUserByID(ctx, actorUID); err == nil {
		actorName = actor.DisplayName
	}
	_ = s.notify.Notify(ctx, NotifyInput{
		UserID:      user.ID.String(),
		ActorID:     actorUID.String(),
		Kind:        domain.NotificationChannelInvite,
		WorkspaceID: ch.WorkspaceID.String(),
		ChannelID:   ch.ID.String(),
		Body:        fmt.Sprintf("%s invited you to join #%s", actorName, ch.Name),
	})
	return nil
}

func (s *ChannelService) RemoveMember(ctx context.Context, actorID, channelID, memberUserID string) error {
	ch, _, err := s.requirePrivateChannelInvite(ctx, actorID, channelID)
	if err != nil {
		return err
	}
	if ch.Kind == "dm" {
		return ErrInvalidInput
	}
	mid, err := uuid.Parse(memberUserID)
	if err != nil {
		return ErrNotFound
	}
	if err := s.store.RemoveChannelMember(ctx, ch.ID, mid); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// CanAccessChannel returns the channel if the user may read/write it.
func (s *ChannelService) CanAccessChannel(ctx context.Context, userID, channelID string) (*domain.Channel, error) {
	row, uid, err := s.loadChannelAccess(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	_ = uid
	ch := row.ToDomain()
	return &ch, nil
}

// ListMembers returns people who can access the channel.
// Public channels include all workspace members; private channels include channel members only.
func (s *ChannelService) ListMembers(
	ctx context.Context, userID, channelID string,
) ([]domain.WorkspaceMember, error) {
	row, _, err := s.loadChannelAccess(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	members, err := s.store.ListWorkspaceMembers(ctx, row.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if row.IsPrivate {
		ids, err := s.store.ListChannelMemberIDs(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		allowed := make(map[uuid.UUID]struct{}, len(ids))
		for _, id := range ids {
			allowed[id] = struct{}{}
		}
		filtered := make([]postgres.WorkspaceMemberInfo, 0, len(ids))
		for _, member := range members {
			if _, ok := allowed[member.UserID]; ok {
				filtered = append(filtered, member)
			}
		}
		members = filtered
	}
	out := make([]domain.WorkspaceMember, 0, len(members))
	for _, member := range members {
		out = append(out, *toWorkspaceMember(member))
	}
	return out, nil
}

func (s *ChannelService) requireChannelManage(
	ctx context.Context, userID, channelID string,
) (postgres.ChannelRow, uuid.UUID, error) {
	row, uid, err := s.loadChannelAccess(ctx, userID, channelID)
	if err != nil {
		return postgres.ChannelRow{}, uuid.Nil, err
	}
	ok, role, err := s.store.IsWorkspaceMember(ctx, row.WorkspaceID, uid)
	if err != nil {
		return postgres.ChannelRow{}, uuid.Nil, err
	}
	if !ok {
		return postgres.ChannelRow{}, uuid.Nil, ErrNotFound
	}
	if canManageWorkspace(domain.WorkspaceRole(role)) || row.CreatedBy == uid {
		return row, uid, nil
	}
	return postgres.ChannelRow{}, uuid.Nil, ErrForbidden
}

func (s *ChannelService) requirePrivateChannelInvite(
	ctx context.Context, userID, channelID string,
) (postgres.ChannelRow, uuid.UUID, error) {
	row, uid, err := s.loadChannelAccess(ctx, userID, channelID)
	if err != nil {
		return postgres.ChannelRow{}, uuid.Nil, err
	}
	if !row.IsPrivate {
		return postgres.ChannelRow{}, uuid.Nil, ErrInvalidInput
	}
	ok, role, err := s.store.IsWorkspaceMember(ctx, row.WorkspaceID, uid)
	if err != nil {
		return postgres.ChannelRow{}, uuid.Nil, err
	}
	if !ok {
		return postgres.ChannelRow{}, uuid.Nil, ErrNotFound
	}
	if canManageWorkspace(domain.WorkspaceRole(role)) {
		return row, uid, nil
	}
	isMember, err := s.store.IsChannelMember(ctx, row.ID, uid)
	if err != nil {
		return postgres.ChannelRow{}, uuid.Nil, err
	}
	if !isMember {
		return postgres.ChannelRow{}, uuid.Nil, ErrForbidden
	}
	return row, uid, nil
}

func (s *ChannelService) loadChannelAccess(
	ctx context.Context, userID, channelID string,
) (postgres.ChannelRow, uuid.UUID, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return postgres.ChannelRow{}, uuid.Nil, ErrInvalidInput
	}
	cid, err := uuid.Parse(channelID)
	if err != nil {
		return postgres.ChannelRow{}, uuid.Nil, ErrNotFound
	}
	row, err := s.store.GetChannel(ctx, cid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return postgres.ChannelRow{}, uuid.Nil, ErrNotFound
		}
		return postgres.ChannelRow{}, uuid.Nil, err
	}
	ok, _, err := s.store.IsWorkspaceMember(ctx, row.WorkspaceID, uid)
	if err != nil {
		return postgres.ChannelRow{}, uuid.Nil, err
	}
	if !ok {
		return postgres.ChannelRow{}, uuid.Nil, ErrNotFound
	}
	if row.IsPrivate {
		isMember, err := s.store.IsChannelMember(ctx, row.ID, uid)
		if err != nil {
			return postgres.ChannelRow{}, uuid.Nil, err
		}
		if !isMember {
			return postgres.ChannelRow{}, uuid.Nil, ErrNotFound
		}
	}
	return row, uid, nil
}

func normaliseChannelName(raw string) (string, error) {
	name := strings.TrimSpace(strings.ToLower(raw))
	name = strings.TrimPrefix(name, "#")
	if name == "" || len(name) > 80 || !channelNamePattern.MatchString(name) {
		return "", ErrChannelName
	}
	return name, nil
}
