package service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/id"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type chatMemStore struct {
	mu             sync.Mutex
	usersByEmail   map[string]postgres.UserRow
	usersByID      map[uuid.UUID]postgres.UserRow
	workspaces     map[uuid.UUID]postgres.WorkspaceRow
	members        map[uuid.UUID]map[uuid.UUID]string
	handles        map[uuid.UUID]map[uuid.UUID]string
	aliases        map[uuid.UUID]map[string]uuid.UUID
	channelsByWS   map[uuid.UUID][]postgres.ChannelRow
	channelsByID   map[uuid.UUID]postgres.ChannelRow
	channelMembers map[uuid.UUID]map[uuid.UUID]string
	dmPairs        map[string]uuid.UUID
	invitesByToken map[string]postgres.WorkspaceInviteRow
	invites        map[uuid.UUID]postgres.WorkspaceInviteRow
	messages       map[uuid.UUID]postgres.MessageRow
	messagesByCh   map[uuid.UUID][]uuid.UUID
	scheduled      map[uuid.UUID]postgres.ScheduledMessageRow
	scheduledByCh  map[uuid.UUID][]uuid.UUID
	notifications  map[uuid.UUID]postgres.NotificationRow
	notifyLevels   map[uuid.UUID]map[uuid.UUID]domain.ChannelNotificationLevel
	linkPreviews   map[uuid.UUID]postgres.LinkPreviewRow
	// readState[userID][channelID] = last read message id; missing inner key means no row.
	// A present key with nil value means unread from the start of the channel.
	readState map[uuid.UUID]map[uuid.UUID]*uuid.UUID
	// reactions[messageID][emoji][userID] = createdAt
	reactions map[uuid.UUID]map[string]map[uuid.UUID]time.Time
	memberOrigins map[uuid.UUID]map[uuid.UUID]memberOrigin
}

func newChatMemStore() *chatMemStore {
	return &chatMemStore{
		usersByEmail:   map[string]postgres.UserRow{},
		usersByID:      map[uuid.UUID]postgres.UserRow{},
		workspaces:     map[uuid.UUID]postgres.WorkspaceRow{},
		members:        map[uuid.UUID]map[uuid.UUID]string{},
		handles:        map[uuid.UUID]map[uuid.UUID]string{},
		aliases:        map[uuid.UUID]map[string]uuid.UUID{},
		channelsByWS:   map[uuid.UUID][]postgres.ChannelRow{},
		channelsByID:   map[uuid.UUID]postgres.ChannelRow{},
		channelMembers: map[uuid.UUID]map[uuid.UUID]string{},
		dmPairs:        map[string]uuid.UUID{},
		invitesByToken: map[string]postgres.WorkspaceInviteRow{},
		invites:        map[uuid.UUID]postgres.WorkspaceInviteRow{},
		messages:       map[uuid.UUID]postgres.MessageRow{},
		messagesByCh:   map[uuid.UUID][]uuid.UUID{},
		scheduled:      map[uuid.UUID]postgres.ScheduledMessageRow{},
		scheduledByCh:  map[uuid.UUID][]uuid.UUID{},
		notifications:  map[uuid.UUID]postgres.NotificationRow{},
		notifyLevels:   map[uuid.UUID]map[uuid.UUID]domain.ChannelNotificationLevel{},
		linkPreviews:   map[uuid.UUID]postgres.LinkPreviewRow{},
		readState:      map[uuid.UUID]map[uuid.UUID]*uuid.UUID{},
		reactions:      map[uuid.UUID]map[string]map[uuid.UUID]time.Time{},
	}
}

func (m *chatMemStore) allocateHandleLocked(workspaceID, userID uuid.UUID, displayName string) string {
	base := postgres.BaseHandle(displayName)
	if m.handles[workspaceID] == nil {
		m.handles[workspaceID] = map[uuid.UUID]string{}
	}
	used := map[string]struct{}{}
	for _, h := range m.handles[workspaceID] {
		used[h] = struct{}{}
	}
	candidate := base
	for i := 0; i < 50; i++ {
		if _, ok := used[candidate]; !ok {
			m.handles[workspaceID][userID] = candidate
			return candidate
		}
		candidate = fmt.Sprintf("%s_%d", base, i+2)
	}
	fallback := fmt.Sprintf("member_%s", userID.String()[:8])
	m.handles[workspaceID][userID] = fallback
	return fallback
}

func (m *chatMemStore) seedUser(email, displayName string) postgres.UserRow {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	row := postgres.UserRow{
		ID: id.New(), Email: email, DisplayName: displayName,
		PasswordHash: "x", NotificationLevel: string(domain.NotifyMentions),
		CreatedAt: now, UpdatedAt: now,
	}
	m.usersByEmail[email] = row
	m.usersByID[row.ID] = row
	return row
}

func (m *chatMemStore) ListUserNotificationLevels(
	_ context.Context, userIDs []uuid.UUID,
) (map[uuid.UUID]domain.NotificationLevel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[uuid.UUID]domain.NotificationLevel{}
	for _, id := range userIDs {
		row, ok := m.usersByID[id]
		if !ok {
			continue
		}
		level, ok := domain.ParseNotificationLevel(row.NotificationLevel)
		if !ok {
			level = domain.NotifyMentions
		}
		out[id] = level
	}
	return out, nil
}

func (m *chatMemStore) setUserNotificationLevel(userID uuid.UUID, level domain.NotificationLevel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.usersByID[userID]
	if !ok {
		return
	}
	row.NotificationLevel = string(level)
	m.usersByID[userID] = row
	m.usersByEmail[row.Email] = row
}

func (m *chatMemStore) seedWorkspace(owner uuid.UUID, name string) (postgres.WorkspaceRow, postgres.ChannelRow) {
	m.mu.Lock()
	defer m.mu.Unlock()
	wsID := id.New()
	ws := postgres.WorkspaceRow{
		ID: wsID, Name: name, Slug: name, CreatedBy: owner, Role: "owner",
	}
	ch := postgres.ChannelRow{
		ID: id.New(), WorkspaceID: wsID, Name: "general", Kind: "channel", CreatedBy: owner,
	}
	m.workspaces[wsID] = ws
	m.members[wsID] = map[uuid.UUID]string{owner: "owner"}
	displayName := "owner"
	if u, ok := m.usersByID[owner]; ok && u.DisplayName != "" {
		displayName = u.DisplayName
	}
	m.allocateHandleLocked(wsID, owner, displayName)
	m.channelsByWS[wsID] = []postgres.ChannelRow{ch}
	m.channelsByID[ch.ID] = ch
	return ws, ch
}

func (m *chatMemStore) GetUserByEmail(_ context.Context, email string) (postgres.UserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.usersByEmail[email]
	if !ok {
		return postgres.UserRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *chatMemStore) GetUserByID(_ context.Context, userID uuid.UUID) (postgres.UserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.usersByID[userID]
	if !ok {
		return postgres.UserRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *chatMemStore) GetWorkspaceForUser(_ context.Context, workspaceID, userID uuid.UUID) (postgres.WorkspaceRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	members, ok := m.members[workspaceID]
	if !ok {
		return postgres.WorkspaceRow{}, postgres.ErrNotFound
	}
	role, ok := members[userID]
	if !ok {
		return postgres.WorkspaceRow{}, postgres.ErrNotFound
	}
	ws := m.workspaces[workspaceID]
	ws.Role = role
	return ws, nil
}

func (m *chatMemStore) IsWorkspaceMember(_ context.Context, workspaceID, userID uuid.UUID) (bool, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	role, ok := m.members[workspaceID][userID]
	if !ok {
		return false, "", nil
	}
	return true, role, nil
}

func (m *chatMemStore) AddWorkspaceMember(_ context.Context, workspaceID, userID uuid.UUID, role domain.WorkspaceRole) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members[workspaceID] == nil {
		m.members[workspaceID] = map[uuid.UUID]string{}
	}
	if _, ok := m.members[workspaceID][userID]; !ok {
		m.members[workspaceID][userID] = string(role)
		displayName := "member"
		if u, ok := m.usersByID[userID]; ok && u.DisplayName != "" {
			displayName = u.DisplayName
		}
		m.allocateHandleLocked(workspaceID, userID, displayName)
	}
	return nil
}

func (m *chatMemStore) CreateChannel(
	_ context.Context, workspaceID, createdBy uuid.UUID, name, topic string, isPrivate bool,
) (postgres.ChannelRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ch := range m.channelsByWS[workspaceID] {
		if ch.Name == name {
			return postgres.ChannelRow{}, postgres.ErrChannelNameConflict
		}
	}
	now := time.Now().UTC()
	row := postgres.ChannelRow{
		ID: id.New(), WorkspaceID: workspaceID, Name: name, Topic: topic,
		IsPrivate: isPrivate, Kind: "channel", CreatedBy: createdBy, CreatedAt: now, UpdatedAt: now,
	}
	m.channelsByWS[workspaceID] = append(m.channelsByWS[workspaceID], row)
	m.channelsByID[row.ID] = row
	if isPrivate {
		m.channelMembers[row.ID] = map[uuid.UUID]string{createdBy: string(domain.ChannelMemberRoleAdmin)}
	}
	return row, nil
}

func (m *chatMemStore) GetChannel(_ context.Context, channelID uuid.UUID) (postgres.ChannelRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.channelsByID[channelID]
	if !ok {
		return postgres.ChannelRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func chatDMPairKey(workspaceID, a, b uuid.UUID) string {
	left, right := a, b
	if a.String() > b.String() {
		left, right = b, a
	}
	return workspaceID.String() + "|" + left.String() + "|" + right.String()
}

func (m *chatMemStore) enrichDMPeer(row postgres.ChannelRow, viewerID uuid.UUID) postgres.ChannelRow {
	key := ""
	for k, chID := range m.dmPairs {
		if chID == row.ID {
			key = k
			break
		}
	}
	if key == "" {
		return row
	}
	parts := strings.Split(key, "|")
	if len(parts) != 3 {
		return row
	}
	left, _ := uuid.Parse(parts[1])
	right, _ := uuid.Parse(parts[2])
	peerID := right
	if viewerID == right {
		peerID = left
	}
	row.PeerUserID = &peerID
	if peer, ok := m.usersByID[peerID]; ok {
		row.PeerDisplayName = peer.DisplayName
		row.PeerHasAvatar = peer.HasAvatar
		row.PeerAvatarUpdatedAt = peer.AvatarUpdatedAt
	}
	if handles := m.handles[row.WorkspaceID]; handles != nil {
		row.PeerHandle = handles[peerID]
	}
	return row
}

func (m *chatMemStore) GetDMChannelForUser(
	_ context.Context, channelID, viewerID uuid.UUID,
) (postgres.ChannelRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.channelsByID[channelID]
	if !ok || row.Kind != "dm" {
		return postgres.ChannelRow{}, postgres.ErrNotFound
	}
	if _, ok := m.channelMembers[channelID][viewerID]; !ok {
		return postgres.ChannelRow{}, postgres.ErrNotFound
	}
	return m.enrichDMPeer(row, viewerID), nil
}

func (m *chatMemStore) FindDMChannel(
	_ context.Context, workspaceID, userA, userB uuid.UUID,
) (postgres.ChannelRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	chID, ok := m.dmPairs[chatDMPairKey(workspaceID, userA, userB)]
	if !ok {
		return postgres.ChannelRow{}, postgres.ErrNotFound
	}
	row, ok := m.channelsByID[chID]
	if !ok {
		return postgres.ChannelRow{}, postgres.ErrNotFound
	}
	return m.enrichDMPeer(row, userA), nil
}

func (m *chatMemStore) CreateDMChannel(
	_ context.Context, workspaceID, createdBy, peerID uuid.UUID,
) (postgres.ChannelRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatDMPairKey(workspaceID, createdBy, peerID)
	if chID, ok := m.dmPairs[key]; ok {
		return m.enrichDMPeer(m.channelsByID[chID], createdBy), nil
	}
	left, right := createdBy, peerID
	if createdBy.String() > peerID.String() {
		left, right = peerID, createdBy
	}
	now := time.Now().UTC()
	row := postgres.ChannelRow{
		ID: id.New(), WorkspaceID: workspaceID,
		Name: "dm_" + left.String() + "_" + right.String(),
		IsPrivate: true, Kind: "dm", CreatedBy: createdBy, CreatedAt: now, UpdatedAt: now,
	}
	m.channelsByWS[workspaceID] = append(m.channelsByWS[workspaceID], row)
	m.channelsByID[row.ID] = row
	m.channelMembers[row.ID] = map[uuid.UUID]string{
		createdBy: string(domain.ChannelMemberRoleMember),
		peerID:    string(domain.ChannelMemberRoleMember),
	}
	m.dmPairs[key] = row.ID
	return m.enrichDMPeer(row, createdBy), nil
}

func (m *chatMemStore) UpdateChannel(
	_ context.Context, channelID, updatedBy uuid.UUID, name, topic string, isPrivate bool,
) (postgres.ChannelRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.channelsByID[channelID]
	if !ok {
		return postgres.ChannelRow{}, postgres.ErrNotFound
	}
	row.Name = name
	row.Topic = topic
	row.IsPrivate = isPrivate
	row.UpdatedAt = time.Now().UTC()
	m.channelsByID[channelID] = row
	list := m.channelsByWS[row.WorkspaceID]
	for i, ch := range list {
		if ch.ID == channelID {
			list[i] = row
		}
	}
	m.channelsByWS[row.WorkspaceID] = list
	if isPrivate {
		if m.channelMembers[channelID] == nil {
			m.channelMembers[channelID] = map[uuid.UUID]string{}
		}
		m.channelMembers[channelID][updatedBy] = string(domain.ChannelMemberRoleAdmin)
		m.channelMembers[channelID][row.CreatedBy] = string(domain.ChannelMemberRoleAdmin)
	}
	return row, nil
}

func (m *chatMemStore) DeleteChannel(_ context.Context, channelID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.channelsByID[channelID]
	if !ok {
		return postgres.ErrNotFound
	}
	delete(m.channelsByID, channelID)
	delete(m.channelMembers, channelID)
	filtered := make([]postgres.ChannelRow, 0)
	for _, ch := range m.channelsByWS[row.WorkspaceID] {
		if ch.ID != channelID {
			filtered = append(filtered, ch)
		}
	}
	m.channelsByWS[row.WorkspaceID] = filtered
	return nil
}

func (m *chatMemStore) ListChannelsForWorkspace(_ context.Context, workspaceID, userID uuid.UUID) ([]postgres.ChannelRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.members[workspaceID][userID]; !ok {
		return nil, postgres.ErrNotFound
	}
	var out []postgres.ChannelRow
	for _, ch := range m.channelsByWS[workspaceID] {
		include := !ch.IsPrivate
		if ch.IsPrivate {
			_, include = m.channelMembers[ch.ID][userID]
		}
		if !include {
			continue
		}
		m.ensureChannelReadBaselineLocked(userID, ch.ID)
		row := ch
		if row.Kind == "dm" {
			row = m.enrichDMPeer(row, userID)
		}
		row.UnreadCount, row.FirstUnreadMessageID = m.unreadSummaryLocked(userID, ch.ID)
		out = append(out, row)
	}
	return out, nil
}

func (m *chatMemStore) ListChannelMemberIDs(_ context.Context, channelID uuid.UUID) ([]uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []uuid.UUID
	for userID := range m.channelMembers[channelID] {
		out = append(out, userID)
	}
	return out, nil
}

func (m *chatMemStore) GetChannelNotificationLevel(
	_ context.Context, userID, channelID uuid.UUID,
) (domain.ChannelNotificationLevel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if levels := m.notifyLevels[channelID]; levels != nil {
		if level, ok := levels[userID]; ok {
			return level, nil
		}
	}
	return domain.ChannelNotifyMentions, nil
}

func (m *chatMemStore) UpsertChannelNotificationLevel(
	_ context.Context, userID, channelID uuid.UUID, level domain.ChannelNotificationLevel,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.notifyLevels[channelID] == nil {
		m.notifyLevels[channelID] = map[uuid.UUID]domain.ChannelNotificationLevel{}
	}
	m.notifyLevels[channelID][userID] = level
	return nil
}

func (m *chatMemStore) ListChannelNotificationLevels(
	_ context.Context, channelID uuid.UUID,
) (map[uuid.UUID]domain.ChannelNotificationLevel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[uuid.UUID]domain.ChannelNotificationLevel{}
	for userID, level := range m.notifyLevels[channelID] {
		out[userID] = level
	}
	return out, nil
}

func (m *chatMemStore) IsChannelMember(_ context.Context, channelID, userID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.channelMembers[channelID][userID]
	return ok, nil
}

func (m *chatMemStore) AddChannelMember(_ context.Context, channelID, userID uuid.UUID, role domain.ChannelMemberRole) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.channelMembers[channelID] == nil {
		m.channelMembers[channelID] = map[uuid.UUID]string{}
	}
	m.channelMembers[channelID][userID] = string(role)
	return nil
}

func (m *chatMemStore) RemoveChannelMember(_ context.Context, channelID, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.channelMembers[channelID][userID]; !ok {
		return postgres.ErrNotFound
	}
	delete(m.channelMembers[channelID], userID)
	return nil
}

func (m *chatMemStore) CreateWorkspaceInvite(
	_ context.Context, workspaceID, invitedBy uuid.UUID, email, token string, role domain.WorkspaceRole, expiresAt time.Time,
) (postgres.WorkspaceInviteRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inv := range m.invites {
		if inv.WorkspaceID == workspaceID && inv.Email == email && inv.AcceptedAt == nil {
			return postgres.WorkspaceInviteRow{}, postgres.ErrInviteConflict
		}
	}
	now := time.Now().UTC()
	row := postgres.WorkspaceInviteRow{
		ID: id.New(), WorkspaceID: workspaceID, Email: email, Token: token,
		Role: string(role), InvitedBy: invitedBy, ExpiresAt: expiresAt,
		CreatedAt: now, UpdatedAt: now,
	}
	m.invites[row.ID] = row
	m.invitesByToken[token] = row
	return row, nil
}

func (m *chatMemStore) ListPendingWorkspaceInvites(_ context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceInviteRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.WorkspaceInviteRow
	for _, inv := range m.invites {
		if inv.WorkspaceID == workspaceID && inv.AcceptedAt == nil {
			out = append(out, inv)
		}
	}
	return out, nil
}

func (m *chatMemStore) GetWorkspaceInviteByToken(_ context.Context, token string) (postgres.WorkspaceInviteRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.invitesByToken[token]
	if !ok {
		return postgres.WorkspaceInviteRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *chatMemStore) AcceptWorkspaceInvite(_ context.Context, inviteID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.invites[inviteID]
	if !ok {
		return postgres.ErrNotFound
	}
	now := time.Now().UTC()
	row.AcceptedAt = &now
	m.invites[inviteID] = row
	m.invitesByToken[row.Token] = row
	return nil
}

func (m *chatMemStore) InsertMessage(
	_ context.Context, channelID, authorID uuid.UUID, body, contentType string,
) (postgres.MessageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	author := m.usersByID[authorID]
	handle := ""
	if ch, ok := m.channelsByID[channelID]; ok {
		handle = m.handles[ch.WorkspaceID][authorID]
	}
	row := postgres.MessageRow{
		ID: id.New(), ChannelID: channelID, AuthorID: authorID,
		AuthorName: author.DisplayName, AuthorHandle: handle,
		AuthorHasAvatar: author.HasAvatar, AuthorAvatarUpdated: author.AvatarUpdatedAt,
		Body: body, ContentType: contentType, CreatedAt: now, UpdatedAt: now,
	}
	m.messages[row.ID] = row
	m.messagesByCh[channelID] = append(m.messagesByCh[channelID], row.ID)
	return row, nil
}

func (m *chatMemStore) ListMessages(
	_ context.Context, channelID uuid.UUID, before *time.Time, beforeID *uuid.UUID, after *time.Time, limit int,
) ([]postgres.MessageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	ids := m.messagesByCh[channelID]
	var out []postgres.MessageRow
	for i := len(ids) - 1; i >= 0; i-- {
		row := m.messages[ids[i]]
		if after != nil && row.CreatedAt.Before(*after) {
			continue
		}
		if before != nil && beforeID != nil {
			if !(row.CreatedAt.Before(*before) || (row.CreatedAt.Equal(*before) && row.ID.String() < beforeID.String())) {
				continue
			}
		}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *chatMemStore) GetMessage(_ context.Context, messageID uuid.UUID) (postgres.MessageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.messages[messageID]
	if !ok {
		return postgres.MessageRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *chatMemStore) UpdateMessageBody(
	_ context.Context, messageID uuid.UUID, body string,
) (postgres.MessageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.messages[messageID]
	if !ok {
		return postgres.MessageRow{}, postgres.ErrNotFound
	}
	row.Body = body
	row.UpdatedAt = time.Now().UTC()
	m.messages[messageID] = row
	return row, nil
}

func (m *chatMemStore) DeleteMessage(_ context.Context, messageID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.messages[messageID]
	if !ok {
		return postgres.ErrNotFound
	}
	delete(m.messages, messageID)
	ids := m.messagesByCh[row.ChannelID]
	next := ids[:0]
	for _, id := range ids {
		if id != messageID {
			next = append(next, id)
		}
	}
	m.messagesByCh[row.ChannelID] = next
	return nil
}

func (m *chatMemStore) UpsertChannelReadState(
	_ context.Context, userID, channelID uuid.UUID, lastReadMessageID *uuid.UUID,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.readState[userID] == nil {
		m.readState[userID] = map[uuid.UUID]*uuid.UUID{}
	}
	m.readState[userID][channelID] = lastReadMessageID
	return nil
}

func (m *chatMemStore) GetChannelReadState(
	_ context.Context, userID, channelID uuid.UUID,
) (postgres.ChannelReadState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := postgres.ChannelReadState{UserID: userID, ChannelID: channelID}
	byChannel, ok := m.readState[userID]
	if !ok {
		return state, nil
	}
	last, ok := byChannel[channelID]
	if !ok {
		return state, nil
	}
	state.HasRow = true
	state.LastReadMessageID = last
	return state, nil
}

func (m *chatMemStore) GetPreviousMessageID(
	_ context.Context, channelID, messageID uuid.UUID,
) (*uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[messageID]
	if !ok || msg.ChannelID != channelID {
		return nil, postgres.ErrNotFound
	}
	var prev *uuid.UUID
	for _, id := range m.messagesByCh[channelID] {
		row := m.messages[id]
		if row.CreatedAt.After(msg.CreatedAt) {
			continue
		}
		if row.CreatedAt.Equal(msg.CreatedAt) && row.ID.String() >= msg.ID.String() {
			continue
		}
		idCopy := row.ID
		if prev == nil {
			prev = &idCopy
			continue
		}
		existing := m.messages[*prev]
		if row.CreatedAt.After(existing.CreatedAt) ||
			(row.CreatedAt.Equal(existing.CreatedAt) && row.ID.String() > existing.ID.String()) {
			prev = &idCopy
		}
	}
	return prev, nil
}

func (m *chatMemStore) GetLatestMessageID(_ context.Context, channelID uuid.UUID) (*uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := m.messagesByCh[channelID]
	if len(ids) == 0 {
		return nil, nil
	}
	latest := ids[len(ids)-1]
	for _, id := range ids {
		row := m.messages[id]
		cur := m.messages[latest]
		if row.CreatedAt.After(cur.CreatedAt) ||
			(row.CreatedAt.Equal(cur.CreatedAt) && row.ID.String() > cur.ID.String()) {
			latest = id
		}
	}
	out := latest
	return &out, nil
}

func (m *chatMemStore) GetChannelUnreadSummary(
	_ context.Context, userID, channelID uuid.UUID,
) (int, *uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count, first := m.unreadSummaryLocked(userID, channelID)
	return count, first, nil
}

func (m *chatMemStore) AddMessageReaction(
	_ context.Context, messageID, userID uuid.UUID, emoji string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.messages[messageID]; !ok {
		return postgres.ErrNotFound
	}
	if m.reactions[messageID] == nil {
		m.reactions[messageID] = map[string]map[uuid.UUID]time.Time{}
	}
	if m.reactions[messageID][emoji] == nil {
		m.reactions[messageID][emoji] = map[uuid.UUID]time.Time{}
	}
	if _, ok := m.reactions[messageID][emoji][userID]; !ok {
		m.reactions[messageID][emoji][userID] = time.Now().UTC()
	}
	return nil
}

func (m *chatMemStore) RemoveMessageReaction(
	_ context.Context, messageID, userID uuid.UUID, emoji string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	users := m.reactions[messageID][emoji]
	if users == nil {
		return postgres.ErrNotFound
	}
	if _, ok := users[userID]; !ok {
		return postgres.ErrNotFound
	}
	delete(users, userID)
	if len(users) == 0 {
		delete(m.reactions[messageID], emoji)
	}
	return nil
}

func (m *chatMemStore) HasMessageReaction(
	_ context.Context, messageID, userID uuid.UUID, emoji string,
) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.reactions[messageID][emoji][userID]
	return ok, nil
}

func (m *chatMemStore) ListReactionsForMessages(
	_ context.Context, messageIDs []uuid.UUID,
) ([]postgres.MessageReactionRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.MessageReactionRow
	for _, messageID := range messageIDs {
		for emoji, users := range m.reactions[messageID] {
			for userID, createdAt := range users {
				out = append(out, postgres.MessageReactionRow{
					MessageID: messageID,
					UserID:    userID,
					Emoji:     emoji,
					CreatedAt: createdAt,
				})
			}
		}
	}
	return out, nil
}

func (m *chatMemStore) ensureChannelReadBaselineLocked(userID, channelID uuid.UUID) {
	byChannel, ok := m.readState[userID]
	if ok {
		if _, has := byChannel[channelID]; has {
			return
		}
	} else {
		m.readState[userID] = map[uuid.UUID]*uuid.UUID{}
	}
	ids := m.messagesByCh[channelID]
	var latest *uuid.UUID
	if len(ids) > 0 {
		idCopy := ids[len(ids)-1]
		latest = &idCopy
	}
	m.readState[userID][channelID] = latest
}

func (m *chatMemStore) unreadSummaryLocked(userID, channelID uuid.UUID) (int, *uuid.UUID) {
	byChannel, ok := m.readState[userID]
	if !ok {
		return 0, nil
	}
	lastRead, ok := byChannel[channelID]
	if !ok {
		return 0, nil
	}
	var anchor *postgres.MessageRow
	if lastRead != nil {
		if row, ok := m.messages[*lastRead]; ok {
			anchor = &row
		}
	}
	var (
		count int
		first *uuid.UUID
	)
	for _, id := range m.messagesByCh[channelID] {
		row := m.messages[id]
		if anchor != nil && postgres.MessageIsAtOrBefore(row, *anchor) {
			continue
		}
		count++
		if first == nil {
			idCopy := row.ID
			first = &idCopy
		}
	}
	return count, first
}

func (m *chatMemStore) InsertScheduledMessage(
	_ context.Context, channelID, authorID uuid.UUID, body, contentType string, sendAt time.Time,
) (postgres.ScheduledMessageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	row := postgres.ScheduledMessageRow{
		ID: id.New(), ChannelID: channelID, AuthorID: authorID, Body: body,
		ContentType: contentType, SendAt: sendAt, Status: string(domain.ScheduledPending),
		CreatedAt: now, UpdatedAt: now,
	}
	m.scheduled[row.ID] = row
	m.scheduledByCh[channelID] = append(m.scheduledByCh[channelID], row.ID)
	return row, nil
}

func (m *chatMemStore) ListScheduledMessages(_ context.Context, channelID uuid.UUID) ([]postgres.ScheduledMessageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.ScheduledMessageRow
	for _, sid := range m.scheduledByCh[channelID] {
		row := m.scheduled[sid]
		if row.Status == string(domain.ScheduledPending) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (m *chatMemStore) GetScheduledMessage(_ context.Context, scheduledID uuid.UUID) (postgres.ScheduledMessageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.scheduled[scheduledID]
	if !ok {
		return postgres.ScheduledMessageRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *chatMemStore) CancelScheduledMessage(_ context.Context, scheduledID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.scheduled[scheduledID]
	if !ok || row.Status != string(domain.ScheduledPending) {
		return postgres.ErrNotFound
	}
	row.Status = string(domain.ScheduledCancelled)
	m.scheduled[scheduledID] = row
	return nil
}

func (m *chatMemStore) ClaimAndPublishDueScheduledMessages(
	_ context.Context, now time.Time, limit int,
) ([]postgres.PublishedScheduledMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	var published []postgres.PublishedScheduledMessage
	for _, row := range m.scheduled {
		if len(published) >= limit {
			break
		}
		if row.Status != string(domain.ScheduledPending) || row.SendAt.After(now) {
			continue
		}
		msgID := id.New()
		msg := postgres.MessageRow{
			ID: msgID, ChannelID: row.ChannelID, AuthorID: row.AuthorID,
			Body: row.Body, ContentType: row.ContentType,
			CreatedAt: now, UpdatedAt: now,
		}
		m.messages[msgID] = msg
		m.messagesByCh[row.ChannelID] = append(m.messagesByCh[row.ChannelID], msgID)
		row.Status = string(domain.ScheduledSent)
		row.SentMessageID = &msgID
		m.scheduled[row.ID] = row
		published = append(published, postgres.PublishedScheduledMessage{
			ID: msgID, ChannelID: row.ChannelID, AuthorID: row.AuthorID, Body: row.Body,
		})
	}
	return published, nil
}

func (m *chatMemStore) formerHandlesLocked(workspaceID, userID uuid.UUID) []string {
	var out []string
	for handle, owner := range m.aliases[workspaceID] {
		if owner == userID {
			out = append(out, handle)
		}
	}
	return out
}

func (m *chatMemStore) ListWorkspaceMembers(_ context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceMemberInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.WorkspaceMemberInfo
	for userID := range m.members[workspaceID] {
		info, err := m.workspaceMemberLocked(workspaceID, userID)
		if err != nil {
			continue
		}
		out = append(out, info)
	}
	return out, nil
}

func (m *chatMemStore) CreateNotification(
	_ context.Context, in postgres.CreateNotificationInput,
) (postgres.NotificationRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.notifications == nil {
		m.notifications = map[uuid.UUID]postgres.NotificationRow{}
	}
	now := time.Now().UTC()
	row := postgres.NotificationRow{
		ID: id.New(), UserID: in.UserID, ActorID: in.ActorID, Kind: string(in.Kind),
		WorkspaceID: in.WorkspaceID, ChannelID: in.ChannelID, MessageID: in.MessageID,
		Body: in.Body, CreatedAt: now,
	}
	if in.ActorID != nil {
		if u, ok := m.usersByID[*in.ActorID]; ok {
			row.ActorName = u.DisplayName
		}
	}
	m.notifications[row.ID] = row
	return row, nil
}

func (m *chatMemStore) enrichNotificationChannelLocked(
	row postgres.NotificationRow, viewerID uuid.UUID,
) postgres.NotificationRow {
	if row.ChannelID == nil {
		return row
	}
	ch, ok := m.channelsByID[*row.ChannelID]
	if !ok {
		return row
	}
	if ch.Kind == "dm" {
		row.IsDM = true
		enriched := m.enrichDMPeer(ch, viewerID)
		if enriched.PeerDisplayName != "" {
			row.ChannelName = enriched.PeerDisplayName
		} else {
			row.ChannelName = "Direct message"
		}
		return row
	}
	row.ChannelName = ch.Name
	return row
}

func (m *chatMemStore) ListNotifications(
	_ context.Context, userID uuid.UUID, filter string, limit int,
) ([]postgres.NotificationRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	var out []postgres.NotificationRow
	for _, row := range m.notifications {
		if row.UserID != userID {
			continue
		}
		if filter == "unread" && row.ReadAt != nil {
			continue
		}
		if filter == "mentions" && row.Kind != string(domain.NotificationMention) {
			continue
		}
		out = append(out, m.enrichNotificationChannelLocked(row, userID))
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *chatMemStore) CountUnreadNotifications(_ context.Context, userID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, row := range m.notifications {
		if row.UserID == userID && row.ReadAt == nil {
			n++
		}
	}
	return n, nil
}

func (m *chatMemStore) MarkNotificationRead(_ context.Context, userID, notificationID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.notifications[notificationID]
	if !ok || row.UserID != userID {
		return postgres.ErrNotFound
	}
	if row.ReadAt == nil {
		now := time.Now().UTC()
		row.ReadAt = &now
		m.notifications[notificationID] = row
	}
	return nil
}

func (m *chatMemStore) MarkNotificationsReadForChannel(_ context.Context, userID, channelID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	for id, row := range m.notifications {
		if row.UserID != userID || row.ReadAt != nil {
			continue
		}
		if row.ChannelID == nil || *row.ChannelID != channelID {
			continue
		}
		switch row.Kind {
		case string(domain.NotificationMention),
			string(domain.NotificationMessage),
			string(domain.NotificationReaction):
			row.ReadAt = &now
			m.notifications[id] = row
		}
	}
	return nil
}

func (m *chatMemStore) MarkAllNotificationsRead(_ context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	for id, row := range m.notifications {
		if row.UserID == userID && row.ReadAt == nil {
			row.ReadAt = &now
			m.notifications[id] = row
		}
	}
	return nil
}

func (m *chatMemStore) ListPendingLinkPreviews(
	_ context.Context, olderThan time.Duration, limit int,
) ([]postgres.LinkPreviewRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 {
		limit = 20
	}
	cutoff := time.Now().UTC().Add(-olderThan)
	var out []postgres.LinkPreviewRow
	for _, row := range m.linkPreviews {
		if row.Status != string(domain.LinkPreviewPending) {
			continue
		}
		if row.CreatedAt.After(cutoff) {
			continue
		}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *chatMemStore) InsertLinkPreview(
	_ context.Context, in postgres.InsertLinkPreviewInput,
) (postgres.LinkPreviewRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.linkPreviews {
		if existing.MessageID == in.MessageID && existing.NormalizedURL == in.NormalizedURL {
			return existing, nil
		}
	}
	now := time.Now().UTC()
	row := postgres.LinkPreviewRow{
		ID: id.New(), MessageID: in.MessageID, URL: in.URL, NormalizedURL: in.NormalizedURL,
		Status: string(domain.LinkPreviewPending), CreatedAt: now, UpdatedAt: now,
	}
	m.linkPreviews[row.ID] = row
	return row, nil
}

func (m *chatMemStore) GetLinkPreview(_ context.Context, previewID uuid.UUID) (postgres.LinkPreviewRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.linkPreviews[previewID]
	if !ok {
		return postgres.LinkPreviewRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *chatMemStore) ListLinkPreviewsForMessages(
	_ context.Context, messageIDs []uuid.UUID,
) ([]postgres.LinkPreviewRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	wanted := map[uuid.UUID]struct{}{}
	for _, id := range messageIDs {
		wanted[id] = struct{}{}
	}
	var out []postgres.LinkPreviewRow
	for _, row := range m.linkPreviews {
		if _, ok := wanted[row.MessageID]; ok {
			out = append(out, row)
		}
	}
	return out, nil
}

func (m *chatMemStore) UpdateLinkPreview(_ context.Context, in postgres.UpdateLinkPreviewInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.linkPreviews[in.ID]
	if !ok {
		return postgres.ErrNotFound
	}
	row.Status = string(in.Status)
	if in.Title != "" {
		row.Title = &in.Title
	}
	if in.Description != "" {
		row.Description = &in.Description
	}
	if in.SiteName != "" {
		row.SiteName = &in.SiteName
	}
	if in.ImageURL != "" {
		row.ImageURL = &in.ImageURL
	}
	if in.Error != "" {
		row.Error = &in.Error
	}
	if in.URL != "" {
		row.URL = in.URL
	}
	now := time.Now().UTC()
	row.FetchedAt = &now
	row.UpdatedAt = now
	m.linkPreviews[in.ID] = row
	return nil
}

type noopLinkUnfurl struct{}

func (noopLinkUnfurl) EnqueueLinkUnfurl(context.Context, string) error { return nil }

func TestLinkPreviewQueuedOnMessagePost(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner-link@example.com", "Owner Link")
	_, general := store.seedWorkspace(owner.ID, "link-ws")
	channels := service.NewChannelService(store)
	messages := service.NewMessageService(store, channels).WithLinkUnfurl(noopLinkUnfurl{})

	msg, err := messages.Post(
		context.Background(),
		owner.ID.String(),
		general.ID.String(),
		"Check https://example.com/docs and also https://example.com/docs#section",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.LinkPreviews) != 1 {
		t.Fatalf("expected 1 pending preview, got %+v", msg.LinkPreviews)
	}
	if msg.LinkPreviews[0].Status != domain.LinkPreviewPending {
		t.Fatalf("status = %q", msg.LinkPreviews[0].Status)
	}
	listed, err := messages.List(context.Background(), owner.ID.String(), general.ID.String(), nil, nil, 20)
	if err != nil || len(listed) != 1 || len(listed[0].LinkPreviews) != 1 {
		t.Fatalf("list previews = %+v err=%v", listed, err)
	}
}

func TestNotificationsMentionInviteJoinAndRead(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner-n@example.com", "Owner Name")
	member := store.seedUser("member-n@example.com", "Member Name")
	outsider := store.seedUser("out-n@example.com", "Outsider")
	ws, general := store.seedWorkspace(owner.ID, "notif-ws")
	_ = store.AddWorkspaceMember(context.Background(), ws.ID, member.ID, domain.WorkspaceRoleMember)

	notifications := service.NewNotificationService(store)
	channels := service.NewChannelService(store).WithNotifications(notifications)
	messages := service.NewMessageService(store, channels).WithNotifications(notifications, notifications)
	invites := service.NewInviteService(store, "http://localhost:5173", nil).WithNotifications(notifications)

	_, err := messages.Post(
		context.Background(),
		owner.ID.String(),
		general.ID.String(),
		"Hey @Member Name please look",
	)
	if err != nil {
		t.Fatal(err)
	}
	memberNotes, err := notifications.List(context.Background(), member.ID.String(), "all", 20)
	if err != nil || len(memberNotes) != 1 || memberNotes[0].Kind != domain.NotificationMention {
		t.Fatalf("member notes = %+v err=%v", memberNotes, err)
	}
	ownerNotes, err := notifications.List(context.Background(), owner.ID.String(), "all", 20)
	if err != nil || len(ownerNotes) != 0 {
		t.Fatalf("author should not be notified: %+v err=%v", ownerNotes, err)
	}

	priv, err := channels.Create(context.Background(), owner.ID.String(), ws.ID.String(), "secret", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := channels.AddMember(context.Background(), owner.ID.String(), priv.ID, member.Email); err != nil {
		t.Fatal(err)
	}
	memberNotes, err = notifications.List(context.Background(), member.ID.String(), "unread", 20)
	if err != nil {
		t.Fatal(err)
	}
	foundInvite := false
	for _, n := range memberNotes {
		if n.Kind == domain.NotificationChannelInvite {
			foundInvite = true
		}
	}
	if !foundInvite {
		t.Fatalf("missing channel invite note: %+v", memberNotes)
	}

	added, err := invites.Invite(context.Background(), owner.ID.String(), ws.ID.String(), outsider.Email, "member")
	if err != nil || added.Status != "added" {
		t.Fatalf("invite existing: %+v err=%v", added, err)
	}
	outNotes, err := notifications.List(context.Background(), outsider.ID.String(), "all", 20)
	if err != nil || len(outNotes) != 1 || outNotes[0].Kind != domain.NotificationWorkspaceInvite {
		t.Fatalf("workspace invite note = %+v err=%v", outNotes, err)
	}

	pending, err := invites.Invite(context.Background(), owner.ID.String(), ws.ID.String(), "fresh@example.com", "member")
	if err != nil || pending.Invite == nil {
		t.Fatalf("pending invite: %+v err=%v", pending, err)
	}
	fresh := store.seedUser("fresh@example.com", "Fresh User")
	if _, err := invites.Accept(context.Background(), fresh.ID.String(), pending.Invite.Token); err != nil {
		t.Fatal(err)
	}
	ownerNotes, err = notifications.List(context.Background(), owner.ID.String(), "all", 20)
	if err != nil {
		t.Fatal(err)
	}
	foundJoin := false
	for _, n := range ownerNotes {
		if n.Kind == domain.NotificationWorkspaceJoin {
			foundJoin = true
		}
	}
	if !foundJoin {
		t.Fatalf("missing workspace join note: %+v", ownerNotes)
	}

	if err := notifications.MarkRead(context.Background(), member.ID.String(), memberNotes[0].ID); err != nil {
		t.Fatal(err)
	}
	unread, err := notifications.UnreadCount(context.Background(), member.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if unread < 1 {
		t.Fatalf("expected remaining unread after one mark, got %d", unread)
	}
	if err := notifications.MarkAllRead(context.Background(), member.ID.String()); err != nil {
		t.Fatal(err)
	}
	unread, err = notifications.UnreadCount(context.Background(), member.ID.String())
	if err != nil || unread != 0 {
		t.Fatalf("unread after mark all = %d err=%v", unread, err)
	}

	leaked, err := notifications.List(context.Background(), outsider.ID.String(), "all", 50)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range leaked {
		if n.UserID != outsider.ID.String() {
			t.Fatalf("leaked notification for other user: %+v", n)
		}
	}
}

func TestChannelNotificationLevels(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner-level@example.com", "Owner Level")
	member := store.seedUser("member-level@example.com", "Member Level")
	ws, general := store.seedWorkspace(owner.ID, "level-ws")
	_ = store.AddWorkspaceMember(context.Background(), ws.ID, member.ID, domain.WorkspaceRoleMember)

	notifications := service.NewNotificationService(store)
	channels := service.NewChannelService(store).WithNotifications(notifications)
	messages := service.NewMessageService(store, channels).WithNotifications(notifications, notifications)

	level, err := notifications.GetChannelNotificationLevel(
		context.Background(), member.ID.String(), general.ID.String(),
	)
	if err != nil || level != domain.ChannelNotifyMentions {
		t.Fatalf("default level = %q err=%v", level, err)
	}

	if _, err := notifications.SetChannelNotificationLevel(
		context.Background(), member.ID.String(), general.ID.String(), string(domain.ChannelNotifyNothing),
	); err != nil {
		t.Fatal(err)
	}
	_, err = messages.Post(
		context.Background(),
		owner.ID.String(),
		general.ID.String(),
		"Hey @Member Level muted",
	)
	if err != nil {
		t.Fatal(err)
	}
	notes, err := notifications.List(context.Background(), member.ID.String(), "all", 20)
	if err != nil || len(notes) != 0 {
		t.Fatalf("muted member should get no mention notes: %+v err=%v", notes, err)
	}

	if _, err := notifications.SetChannelNotificationLevel(
		context.Background(), member.ID.String(), general.ID.String(), string(domain.ChannelNotifyAll),
	); err != nil {
		t.Fatal(err)
	}
	store.setUserNotificationLevel(member.ID, domain.NotifyAll)
	_, err = messages.Post(
		context.Background(),
		owner.ID.String(),
		general.ID.String(),
		"plain update with no mention",
	)
	if err != nil {
		t.Fatal(err)
	}
	notes, err = notifications.List(context.Background(), member.ID.String(), "all", 20)
	if err != nil {
		t.Fatal(err)
	}
	foundMessage := false
	for _, n := range notes {
		if n.Kind == domain.NotificationMessage {
			foundMessage = true
		}
	}
	if !foundMessage {
		t.Fatalf("expected message notification for all level: %+v", notes)
	}

	store.setUserNotificationLevel(member.ID, domain.NotifyMentions)
	_, err = messages.Post(
		context.Background(),
		owner.ID.String(),
		general.ID.String(),
		"another plain update while global is mentions",
	)
	if err != nil {
		t.Fatal(err)
	}
	notes, err = notifications.List(context.Background(), member.ID.String(), "all", 50)
	if err != nil {
		t.Fatal(err)
	}
	messageCount := 0
	for _, n := range notes {
		if n.Kind == domain.NotificationMessage {
			messageCount++
		}
	}
	if messageCount != 1 {
		t.Fatalf("global mentions should block channel all; message notes=%d in %+v", messageCount, notes)
	}

	store.setUserNotificationLevel(member.ID, domain.NotifyNothing)
	_, err = messages.Post(
		context.Background(),
		owner.ID.String(),
		general.ID.String(),
		"Hey @Member Level while globally off",
	)
	if err != nil {
		t.Fatal(err)
	}
	notes, err = notifications.List(context.Background(), member.ID.String(), "all", 50)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range notes {
		if strings.Contains(n.Body, "globally off") {
			t.Fatalf("global nothing should silence mentions: %+v", notes)
		}
	}
}

func TestMentionResolvesFormerHandle(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner-alias@example.com", "Owner Alias")
	member := store.seedUser("member-alias@example.com", "Member Alias")
	ws, general := store.seedWorkspace(owner.ID, "alias-ws")
	_ = store.AddWorkspaceMember(context.Background(), ws.ID, member.ID, domain.WorkspaceRoleMember)

	store.mu.Lock()
	oldHandle := store.handles[ws.ID][member.ID]
	store.aliases[ws.ID] = map[string]uuid.UUID{oldHandle: member.ID}
	store.handles[ws.ID][member.ID] = "new_handle"
	store.mu.Unlock()

	notifications := service.NewNotificationService(store)
	channels := service.NewChannelService(store).WithNotifications(notifications)
	messages := service.NewMessageService(store, channels).WithNotifications(notifications, notifications)

	_, err := messages.Post(
		context.Background(),
		owner.ID.String(),
		general.ID.String(),
		"Hey @"+oldHandle+" still you",
	)
	if err != nil {
		t.Fatal(err)
	}
	notes, err := notifications.List(context.Background(), member.ID.String(), "mentions", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected mention via former handle, got %+v", notes)
	}
}

func TestPrivateChannelHiddenFromNonMembers(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner@example.com", "Owner")
	member := store.seedUser("member@example.com", "Member")
	ws, _ := store.seedWorkspace(owner.ID, "acme")
	_ = store.AddWorkspaceMember(context.Background(), ws.ID, member.ID, domain.WorkspaceRoleMember)

	channels := service.NewChannelService(store)

	priv, err := channels.Create(context.Background(), owner.ID.String(), ws.ID.String(), "secret", "", true)
	if err != nil {
		t.Fatal(err)
	}
	_, err = channels.Create(context.Background(), owner.ID.String(), ws.ID.String(), "public", "", false)
	if err != nil {
		t.Fatal(err)
	}

	ownerList, err := store.ListChannelsForWorkspace(context.Background(), ws.ID, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ownerList) != 3 {
		t.Fatalf("owner channels = %d, want 3", len(ownerList))
	}

	memberList, err := store.ListChannelsForWorkspace(context.Background(), ws.ID, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range memberList {
		if ch.ID.String() == priv.ID {
			t.Fatal("private channel visible to non-member")
		}
	}
	if len(memberList) != 2 {
		t.Fatalf("member channels = %d, want 2", len(memberList))
	}
}

func TestChannelPrivacyToggle(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner-privacy@example.com", "Owner")
	member := store.seedUser("member-privacy@example.com", "Member")
	ws, _ := store.seedWorkspace(owner.ID, "privacy-ws")
	_ = store.AddWorkspaceMember(context.Background(), ws.ID, member.ID, domain.WorkspaceRoleMember)
	channels := service.NewChannelService(store)

	pub, err := channels.Create(context.Background(), owner.ID.String(), ws.ID.String(), "open", "", false)
	if err != nil {
		t.Fatal(err)
	}
	memberList, err := store.ListChannelsForWorkspace(context.Background(), ws.ID, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ch := range memberList {
		if ch.ID.String() == pub.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("public channel should be visible")
	}

	updated, err := channels.Update(context.Background(), owner.ID.String(), pub.ID, "open", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.IsPrivate {
		t.Fatal("expected private after update")
	}
	memberList, err = store.ListChannelsForWorkspace(context.Background(), ws.ID, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range memberList {
		if ch.ID.String() == pub.ID {
			t.Fatal("private channel should be hidden from non-members")
		}
	}

	updated, err = channels.Update(context.Background(), owner.ID.String(), pub.ID, "open", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if updated.IsPrivate {
		t.Fatal("expected public after update")
	}
}

func TestChannelSettingsTimelineEvent(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner-settings@example.com", "Owner")
	ws, _ := store.seedWorkspace(owner.ID, "settings-ws")
	channels := service.NewChannelService(store)
	messages := service.NewMessageService(store, channels)

	ch, err := channels.Create(context.Background(), owner.ID.String(), ws.ID.String(), "ops", "", false)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := channels.Update(
		context.Background(),
		owner.ID.String(),
		ch.ID,
		"ops",
		"On-call notes",
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Topic != "On-call notes" || !updated.IsPrivate {
		t.Fatalf("unexpected update: %+v", updated)
	}

	listed, err := messages.List(context.Background(), owner.ID.String(), ch.ID, nil, nil, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 system message, got %d", len(listed))
	}
	if listed[0].ContentType != domain.ContentTypeSystem {
		t.Fatalf("content type = %q", listed[0].ContentType)
	}
	if !strings.Contains(listed[0].Body, "Owner") ||
		!strings.Contains(listed[0].Body, "set the channel topic") ||
		!strings.Contains(listed[0].Body, "made this channel private") {
		t.Fatalf("body = %q", listed[0].Body)
	}
}

func TestInviteExistingAndPendingAccept(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner2@example.com", "Owner")
	existing := store.seedUser("existing@example.com", "Existing")
	ws, _ := store.seedWorkspace(owner.ID, "invite-ws")
	invites := service.NewInviteService(store, "http://localhost:5173", nil)

	added, err := invites.Invite(context.Background(), owner.ID.String(), ws.ID.String(), existing.Email, "member")
	if err != nil {
		t.Fatal(err)
	}
	if added.Status != "added" {
		t.Fatalf("status = %q", added.Status)
	}
	ok, _, err := store.IsWorkspaceMember(context.Background(), ws.ID, existing.ID)
	if err != nil || !ok {
		t.Fatalf("existing should be member: ok=%v err=%v", ok, err)
	}

	pending, err := invites.Invite(context.Background(), owner.ID.String(), ws.ID.String(), "new@example.com", "member")
	if err != nil {
		t.Fatal(err)
	}
	if pending.Status != "invited" || pending.Invite == nil || pending.Invite.Token == "" {
		t.Fatalf("pending = %+v", pending)
	}

	newbie := store.seedUser("new@example.com", "Newbie")
	joined, err := invites.Accept(context.Background(), newbie.ID.String(), pending.Invite.Token)
	if err != nil {
		t.Fatal(err)
	}
	if joined.ID != ws.ID.String() {
		t.Fatalf("joined workspace = %s", joined.ID)
	}
}

type staticEntitlements struct {
	ents domain.Entitlements
}

func (s staticEntitlements) ForWorkspace(context.Context, string) domain.Entitlements {
	return s.ents
}

func TestMessageListHonoursFreeHistoryFloor(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner-history@example.com", "Owner History")
	_, general := store.seedWorkspace(owner.ID, "history-ws")
	channels := service.NewChannelService(store)
	messages := service.NewMessageService(store, channels).WithEntitlements(staticEntitlements{
		ents: domain.FreeCloudEntitlements(),
	})

	oldMsg, err := messages.Post(context.Background(), owner.ID.String(), general.ID.String(), "old")
	if err != nil {
		t.Fatal(err)
	}
	newMsg, err := messages.Post(context.Background(), owner.ID.String(), general.ID.String(), "new")
	if err != nil {
		t.Fatal(err)
	}
	oldID, _ := uuid.Parse(oldMsg.ID)
	store.mu.Lock()
	row := store.messages[oldID]
	row.CreatedAt = time.Now().UTC().AddDate(0, 0, -91)
	store.messages[oldID] = row
	store.mu.Unlock()

	result, err := messages.ListWithMeta(context.Background(), owner.ID.String(), general.ID.String(), nil, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !result.HistoryLimited || result.HistoryRetentionDays == nil || *result.HistoryRetentionDays != 90 {
		t.Fatalf("meta = %+v", result)
	}
	if len(result.Messages) != 1 || result.Messages[0].ID != newMsg.ID {
		t.Fatalf("messages = %+v", result.Messages)
	}
}

func TestMessageAccessAndSchedulePublish(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner3@example.com", "Owner")
	outsider := store.seedUser("out@example.com", "Out")
	ws, general := store.seedWorkspace(owner.ID, "msg-ws")
	channels := service.NewChannelService(store)
	messages := service.NewMessageService(store, channels)

	priv, err := channels.Create(context.Background(), owner.ID.String(), ws.ID.String(), "private", "", true)
	if err != nil {
		t.Fatal(err)
	}
	_ = store.AddWorkspaceMember(context.Background(), ws.ID, outsider.ID, domain.WorkspaceRoleMember)

	_, err = messages.Post(context.Background(), outsider.ID.String(), priv.ID, "nope")
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	msg, err := messages.Post(context.Background(), owner.ID.String(), general.ID.String(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	list, err := messages.List(context.Background(), owner.ID.String(), general.ID.String(), nil, nil, 10)
	if err != nil || len(list) != 1 || list[0].ID != msg.ID {
		t.Fatalf("list = %+v err=%v", list, err)
	}

	sendAt := time.Now().UTC().Add(2 * time.Minute)
	scheduled, err := messages.Schedule(context.Background(), owner.ID.String(), general.ID.String(), "later", sendAt)
	if err != nil {
		t.Fatal(err)
	}
	if scheduled.Status != domain.ScheduledPending {
		t.Fatalf("status = %s", scheduled.Status)
	}

	n, err := messages.PublishDue(context.Background(), sendAt.Add(time.Second), 10)
	if err != nil || n != 1 {
		t.Fatalf("publish n=%d err=%v", n, err)
	}
	list, err = messages.List(context.Background(), owner.ID.String(), general.ID.String(), nil, nil, 10)
	if err != nil || len(list) != 2 {
		t.Fatalf("after publish list=%d err=%v", len(list), err)
	}
}

func TestMessageUpdateAndDeleteOwn(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner-edit@example.com", "Owner Edit")
	member := store.seedUser("member-edit@example.com", "Member Edit")
	ws, general := store.seedWorkspace(owner.ID, "edit-ws")
	_ = store.AddWorkspaceMember(context.Background(), ws.ID, member.ID, domain.WorkspaceRoleMember)
	channels := service.NewChannelService(store)
	messages := service.NewMessageService(store, channels)
	ctx := context.Background()

	msg, err := messages.Post(ctx, owner.ID.String(), general.ID.String(), "original")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := messages.Update(ctx, member.ID.String(), msg.ID, "hijack"); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("expected forbidden for other user, got %v", err)
	}

	updated, err := messages.Update(ctx, owner.ID.String(), msg.ID, "edited body")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Body != "edited body" {
		t.Fatalf("body = %q", updated.Body)
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Fatalf("expected updated_at after created_at")
	}

	if err := messages.Delete(ctx, member.ID.String(), msg.ID); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("expected forbidden delete, got %v", err)
	}
	if err := messages.Delete(ctx, owner.ID.String(), msg.ID); err != nil {
		t.Fatal(err)
	}
	list, err := messages.List(ctx, owner.ID.String(), general.ID.String(), nil, nil, 10)
	if err != nil || len(list) != 0 {
		t.Fatalf("list after delete = %+v err=%v", list, err)
	}
}

func TestMarkMessageUnread(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner-unread@example.com", "Owner Unread")
	member := store.seedUser("member-unread@example.com", "Member Unread")
	ws, general := store.seedWorkspace(owner.ID, "unread-ws")
	_ = store.AddWorkspaceMember(context.Background(), ws.ID, member.ID, domain.WorkspaceRoleMember)
	channels := service.NewChannelService(store)
	messages := service.NewMessageService(store, channels)
	ctx := context.Background()

	first, err := messages.Post(ctx, owner.ID.String(), general.ID.String(), "one")
	if err != nil {
		t.Fatal(err)
	}
	second, err := messages.Post(ctx, owner.ID.String(), general.ID.String(), "two")
	if err != nil {
		t.Fatal(err)
	}
	third, err := messages.Post(ctx, owner.ID.String(), general.ID.String(), "three")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := messages.MarkRead(ctx, member.ID.String(), general.ID.String(), third.ID); err != nil {
		t.Fatal(err)
	}
	_ = first

	unread, err := messages.MarkUnread(ctx, member.ID.String(), general.ID.String(), second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unread.UnreadCount != 2 {
		t.Fatalf("unread count = %d, want 2", unread.UnreadCount)
	}
	if unread.FirstUnreadMessageID != second.ID {
		t.Fatalf("first unread = %s, want %s", unread.FirstUnreadMessageID, second.ID)
	}

	channelList, err := store.ListChannelsForWorkspace(ctx, ws.ID, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, ch := range channelList {
		if ch.ID != general.ID {
			continue
		}
		found = true
		if ch.UnreadCount != 2 {
			t.Fatalf("channel unread = %d", ch.UnreadCount)
		}
		if ch.FirstUnreadMessageID == nil || ch.FirstUnreadMessageID.String() != second.ID {
			t.Fatalf("channel first unread = %v", ch.FirstUnreadMessageID)
		}
	}
	if !found {
		t.Fatal("expected general channel in list")
	}

	caughtUp, err := messages.MarkRead(ctx, member.ID.String(), general.ID.String(), "")
	if err != nil {
		t.Fatal(err)
	}
	if caughtUp.UnreadCount != 0 {
		t.Fatalf("after mark read unread = %d", caughtUp.UnreadCount)
	}

	own, err := messages.Post(ctx, member.ID.String(), general.ID.String(), "mine")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := messages.MarkUnread(ctx, member.ID.String(), general.ID.String(), own.ID); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("expected forbidden for own message, got %v", err)
	}
}

func TestChannelUnreadBaselineAfterList(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner-baseline@example.com", "Owner Baseline")
	member := store.seedUser("member-baseline@example.com", "Member Baseline")
	ws, general := store.seedWorkspace(owner.ID, "baseline-ws")
	_ = store.AddWorkspaceMember(context.Background(), ws.ID, member.ID, domain.WorkspaceRoleMember)
	channels := service.NewChannelService(store)
	messages := service.NewMessageService(store, channels)
	ctx := context.Background()

	if _, err := messages.Post(ctx, owner.ID.String(), general.ID.String(), "history"); err != nil {
		t.Fatal(err)
	}
	listed, err := store.ListChannelsForWorkspace(ctx, ws.ID, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].UnreadCount != 0 {
		t.Fatalf("after baseline list unread = %+v", listed)
	}

	if _, err := messages.Post(ctx, owner.ID.String(), general.ID.String(), "new"); err != nil {
		t.Fatal(err)
	}
	listed, err = store.ListChannelsForWorkspace(ctx, ws.ID, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].UnreadCount != 1 {
		t.Fatalf("after new message unread = %+v", listed)
	}
}

func TestToggleMessageReaction(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner-react@example.com", "Owner React")
	member := store.seedUser("member-react@example.com", "Member React")
	ws, general := store.seedWorkspace(owner.ID, "react-ws")
	_ = store.AddWorkspaceMember(context.Background(), ws.ID, member.ID, domain.WorkspaceRoleMember)
	notifications := service.NewNotificationService(store)
	channels := service.NewChannelService(store)
	messages := service.NewMessageService(store, channels).WithNotifications(notifications, notifications)
	ctx := context.Background()

	msg, err := messages.Post(ctx, owner.ID.String(), general.ID.String(), "react to me")
	if err != nil {
		t.Fatal(err)
	}

	updated, err := messages.ToggleReaction(ctx, member.ID.String(), msg.ID, "thumbsup")
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Reactions) != 1 || updated.Reactions[0].Emoji != "thumbsup" || updated.Reactions[0].Count != 1 {
		t.Fatalf("reactions = %+v", updated.Reactions)
	}
	if !updated.Reactions[0].Reacted {
		t.Fatal("expected reacted=true for actor")
	}

	notes, err := notifications.List(ctx, owner.ID.String(), "unread", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || notes[0].Kind != domain.NotificationReaction {
		t.Fatalf("owner notifications = %+v", notes)
	}
	if notes[0].MessageID != msg.ID {
		t.Fatalf("notification message id = %q", notes[0].MessageID)
	}
	if !strings.Contains(notes[0].Body, "thumbsup") || !strings.Contains(notes[0].Body, "Member React") {
		t.Fatalf("notification body = %q", notes[0].Body)
	}

	listed, err := messages.List(ctx, owner.ID.String(), general.ID.String(), nil, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || len(listed[0].Reactions) != 1 || listed[0].Reactions[0].Reacted {
		t.Fatalf("owner list reactions = %+v", listed[0].Reactions)
	}

	cleared, err := messages.ToggleReaction(ctx, member.ID.String(), msg.ID, "thumbsup")
	if err != nil {
		t.Fatal(err)
	}
	if len(cleared.Reactions) != 0 {
		t.Fatalf("expected no reactions after toggle off, got %+v", cleared.Reactions)
	}

	// Removing a reaction must not create another notification.
	notesAfter, err := notifications.List(ctx, owner.ID.String(), "all", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(notesAfter) != 1 {
		t.Fatalf("expected still 1 notification, got %d", len(notesAfter))
	}

	// Reacting to your own message must not notify.
	if _, err := messages.ToggleReaction(ctx, owner.ID.String(), msg.ID, "heart"); err != nil {
		t.Fatal(err)
	}
	selfNotes, err := notifications.List(ctx, owner.ID.String(), "all", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(selfNotes) != 1 {
		t.Fatalf("expected no self-reaction notification, got %d", len(selfNotes))
	}
}

func TestOpenDMFindOrCreate(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner-dm@example.com", "Owner DM")
	peer := store.seedUser("peer-dm@example.com", "Peer DM")
	outsider := store.seedUser("out-dm@example.com", "Out DM")
	ws, _ := store.seedWorkspace(owner.ID, "dm-ws")
	_ = store.AddWorkspaceMember(context.Background(), ws.ID, peer.ID, domain.WorkspaceRoleMember)
	channels := service.NewChannelService(store)
	messages := service.NewMessageService(store, channels)

	_, err := channels.OpenDM(context.Background(), owner.ID.String(), ws.ID.String(), owner.ID.String())
	if !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("self DM: got %v, want ErrInvalidInput", err)
	}

	_, err = channels.OpenDM(context.Background(), owner.ID.String(), ws.ID.String(), outsider.ID.String())
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("non-member peer: got %v, want ErrNotFound", err)
	}

	first, err := channels.OpenDM(context.Background(), owner.ID.String(), ws.ID.String(), peer.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if !first.IsDM || first.PeerUserID != peer.ID.String() || first.PeerDisplayName != "Peer DM" {
		t.Fatalf("first DM = %+v", first)
	}
	if !first.IsPrivate {
		t.Fatal("expected private DM channel")
	}

	second, err := channels.OpenDM(context.Background(), peer.ID.String(), ws.ID.String(), owner.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected same channel, got %s vs %s", first.ID, second.ID)
	}
	if second.PeerUserID != owner.ID.String() {
		t.Fatalf("peer from other side = %s", second.PeerUserID)
	}

	msg, err := messages.Post(context.Background(), owner.ID.String(), first.ID, "hey")
	if err != nil {
		t.Fatal(err)
	}
	list, err := messages.List(context.Background(), peer.ID.String(), first.ID, nil, nil, 10)
	if err != nil || len(list) != 1 || list[0].ID != msg.ID {
		t.Fatalf("list = %+v err=%v", list, err)
	}

	listed, err := store.ListChannelsForWorkspace(context.Background(), ws.ID, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, ch := range listed {
		if ch.ID.String() == first.ID {
			found = true
			dom := ch.ToDomain()
			if !dom.IsDM || dom.PeerUserID != peer.ID.String() {
				t.Fatalf("listed DM = %+v", dom)
			}
		}
	}
	if !found {
		t.Fatal("expected DM in channel list")
	}

	if _, err := channels.Update(context.Background(), owner.ID.String(), first.ID, "nope", "", true); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("update DM: got %v", err)
	}
	if err := channels.Delete(context.Background(), owner.ID.String(), first.ID); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("delete DM: got %v", err)
	}
}

func TestDMMessageNotifications(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner-dm-note@example.com", "Owner Note")
	peer := store.seedUser("peer-dm-note@example.com", "Peer Note")
	ws, _ := store.seedWorkspace(owner.ID, "dm-note-ws")
	_ = store.AddWorkspaceMember(context.Background(), ws.ID, peer.ID, domain.WorkspaceRoleMember)

	notifications := service.NewNotificationService(store)
	channels := service.NewChannelService(store).WithNotifications(notifications)
	messages := service.NewMessageService(store, channels).WithNotifications(notifications, notifications)

	dm, err := channels.OpenDM(context.Background(), owner.ID.String(), ws.ID.String(), peer.ID.String())
	if err != nil {
		t.Fatal(err)
	}

	// Default account level is mentions; DMs should still notify the peer.
	msg, err := messages.Post(context.Background(), owner.ID.String(), dm.ID, "hello in dm")
	if err != nil {
		t.Fatal(err)
	}

	notes, err := notifications.List(context.Background(), peer.ID.String(), "all", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 DM notification, got %+v", notes)
	}
	if notes[0].Kind != domain.NotificationMessage || notes[0].MessageID != msg.ID {
		t.Fatalf("unexpected note: %+v", notes[0])
	}
	if strings.Contains(notes[0].Body, "#dm_") {
		t.Fatalf("DM notification should not use synthetic channel name: %q", notes[0].Body)
	}

	store.setUserNotificationLevel(peer.ID, domain.NotifyNothing)
	if _, err := messages.Post(context.Background(), owner.ID.String(), dm.ID, "should be silent"); err != nil {
		t.Fatal(err)
	}
	notes, err = notifications.List(context.Background(), peer.ID.String(), "all", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Fatalf("silenced peer should stay at 1 note, got %d", len(notes))
	}
}

func TestMarkReadClearsChannelNotifications(t *testing.T) {
	t.Parallel()
	store := newChatMemStore()
	owner := store.seedUser("owner-read-note@example.com", "Owner Read")
	peer := store.seedUser("peer-read-note@example.com", "Peer Read")
	ws, _ := store.seedWorkspace(owner.ID, "read-note-ws")
	_ = store.AddWorkspaceMember(context.Background(), ws.ID, peer.ID, domain.WorkspaceRoleMember)

	notifications := service.NewNotificationService(store)
	channels := service.NewChannelService(store).WithNotifications(notifications)
	messages := service.NewMessageService(store, channels).WithNotifications(notifications, notifications)

	dm, err := channels.OpenDM(context.Background(), owner.ID.String(), ws.ID.String(), peer.ID.String())
	if err != nil {
		t.Fatal(err)
	}

	msg, err := messages.Post(context.Background(), owner.ID.String(), dm.ID, "hello while away")
	if err != nil {
		t.Fatal(err)
	}

	unread, err := notifications.UnreadCount(context.Background(), peer.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if unread != 1 {
		t.Fatalf("expected 1 unread notification, got %d", unread)
	}

	if _, err := messages.MarkRead(context.Background(), peer.ID.String(), dm.ID, msg.ID); err != nil {
		t.Fatal(err)
	}

	unread, err = notifications.UnreadCount(context.Background(), peer.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if unread != 0 {
		t.Fatalf("expected unread notifications cleared after mark read, got %d", unread)
	}
	notes, err := notifications.List(context.Background(), peer.ID.String(), "unread", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Fatalf("expected no unread notifications, got %+v", notes)
	}
}

func (m *chatMemStore) CountWorkspaceOwners(_ context.Context, workspaceID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, role := range m.members[workspaceID] {
		if role == string(domain.WorkspaceRoleOwner) {
			n++
		}
	}
	return n, nil
}

func (m *chatMemStore) UpdateWorkspaceMemberRole(
	_ context.Context, workspaceID, userID uuid.UUID, role domain.WorkspaceRole,
) (postgres.WorkspaceMemberInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.members[workspaceID][userID]; !ok {
		return postgres.WorkspaceMemberInfo{}, postgres.ErrNotFound
	}
	m.members[workspaceID][userID] = string(role)
	return m.workspaceMemberLocked(workspaceID, userID)
}

func (m *chatMemStore) RemoveWorkspaceMember(_ context.Context, workspaceID, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.members[workspaceID][userID]; !ok {
		return postgres.ErrNotFound
	}
	delete(m.members[workspaceID], userID)
	if m.handles[workspaceID] != nil {
		delete(m.handles[workspaceID], userID)
	}
	return nil
}

func (m *chatMemStore) TransferWorkspaceOwnership(
	_ context.Context, workspaceID, fromOwner, toUser uuid.UUID,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members[workspaceID][fromOwner] != string(domain.WorkspaceRoleOwner) {
		return postgres.ErrNotFound
	}
	if _, ok := m.members[workspaceID][toUser]; !ok {
		return postgres.ErrNotFound
	}
	m.members[workspaceID][fromOwner] = string(domain.WorkspaceRoleMember)
	m.members[workspaceID][toUser] = string(domain.WorkspaceRoleOwner)
	return nil
}

func (m *chatMemStore) DeleteWorkspaceInvite(_ context.Context, workspaceID, inviteID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.invites[inviteID]
	if !ok || row.WorkspaceID != workspaceID || row.AcceptedAt != nil {
		return postgres.ErrNotFound
	}
	delete(m.invites, inviteID)
	delete(m.invitesByToken, row.Token)
	return nil
}

func (m *chatMemStore) FindHomeWorkspaceForUser(_ context.Context, userID uuid.UUID) (uuid.UUID, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for wsID, members := range m.members {
		if members[userID] != string(domain.WorkspaceRoleOwner) {
			continue
		}
		ws, ok := m.workspaces[wsID]
		if !ok {
			continue
		}
		return wsID, ws.Name, nil
	}
	return uuid.Nil, "", postgres.ErrNotFound
}

func (m *chatMemStore) SetWorkspaceMemberOrigin(
	_ context.Context,
	workspaceID, userID uuid.UUID,
	kind string,
	homeWorkspaceID *uuid.UUID,
	homeWorkspaceName string,
	homeServerID, homeWorkspaceRemoteID, homeWorkspaceIconURL string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.members[workspaceID][userID]; !ok {
		return postgres.ErrNotFound
	}
	if m.memberOrigins == nil {
		m.memberOrigins = map[uuid.UUID]map[uuid.UUID]memberOrigin{}
	}
	if m.memberOrigins[workspaceID] == nil {
		m.memberOrigins[workspaceID] = map[uuid.UUID]memberOrigin{}
	}
	m.memberOrigins[workspaceID][userID] = memberOrigin{
		Kind: kind, HomeWorkspaceID: homeWorkspaceID, HomeWorkspaceName: homeWorkspaceName,
		HomeServerID: homeServerID, HomeWorkspaceRemoteID: homeWorkspaceRemoteID,
		HomeWorkspaceIconURL: homeWorkspaceIconURL,
	}
	return nil
}

type memberOrigin struct {
	Kind                  string
	HomeWorkspaceID       *uuid.UUID
	HomeWorkspaceName     string
	HomeServerID          string
	HomeWorkspaceRemoteID string
	HomeWorkspaceIconURL  string
}

func (m *chatMemStore) workspaceMemberLocked(workspaceID, userID uuid.UUID) (postgres.WorkspaceMemberInfo, error) {
	role, ok := m.members[workspaceID][userID]
	if !ok {
		return postgres.WorkspaceMemberInfo{}, postgres.ErrNotFound
	}
	name := ""
	if u, ok := m.usersByID[userID]; ok {
		name = u.DisplayName
	}
	info := postgres.WorkspaceMemberInfo{
		UserID: userID, DisplayName: name, Handle: m.handles[workspaceID][userID],
		FormerHandles: m.formerHandlesLocked(workspaceID, userID), Role: role, Kind: "member",
	}
	if origins := m.memberOrigins[workspaceID]; origins != nil {
		if origin, ok := origins[userID]; ok {
			info.Kind = origin.Kind
			info.HomeWorkspaceID = origin.HomeWorkspaceID
			info.HomeWorkspaceName = origin.HomeWorkspaceName
			info.HomeServerID = origin.HomeServerID
			info.HomeWorkspaceRemoteID = origin.HomeWorkspaceRemoteID
			info.HomeWorkspaceIconURL = origin.HomeWorkspaceIconURL
		}
	}
	return info, nil
}
