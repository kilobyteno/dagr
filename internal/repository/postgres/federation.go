package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type FederatedPeerRow struct {
	ServerID         string
	PublicURL        string
	SigningPublicKey string
	Status           string
	TrustedAt        time.Time
	CreatedAt        time.Time
}

type ConversationPeerRow struct {
	ChannelID     uuid.UUID
	PeerServerID  string
	PeerChannelID string
	CreatedAt     time.Time
}

func (s *Store) UpsertFederatedPeer(
	ctx context.Context, serverID, publicURL, signingPublicKey string,
) (FederatedPeerRow, error) {
	var row FederatedPeerRow
	err := s.pool.QueryRow(ctx, `
		INSERT INTO federated_peers (server_id, public_url, signing_public_key, status)
		VALUES ($1, $2, $3, 'trusted')
		ON CONFLICT (server_id) DO UPDATE
		SET public_url = EXCLUDED.public_url,
			signing_public_key = EXCLUDED.signing_public_key,
			status = 'trusted',
			trusted_at = now()
		RETURNING server_id, public_url, signing_public_key, status, trusted_at, created_at
	`, serverID, publicURL, signingPublicKey).Scan(
		&row.ServerID, &row.PublicURL, &row.SigningPublicKey, &row.Status, &row.TrustedAt, &row.CreatedAt,
	)
	if err != nil {
		return FederatedPeerRow{}, fmt.Errorf("upsert federated peer: %w", err)
	}
	return row, nil
}

func (s *Store) GetFederatedPeer(ctx context.Context, serverID string) (FederatedPeerRow, error) {
	var row FederatedPeerRow
	err := s.pool.QueryRow(ctx, `
		SELECT server_id, public_url, signing_public_key, status, trusted_at, created_at
		FROM federated_peers WHERE server_id = $1
	`, serverID).Scan(
		&row.ServerID, &row.PublicURL, &row.SigningPublicKey, &row.Status, &row.TrustedAt, &row.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return FederatedPeerRow{}, ErrNotFound
	}
	if err != nil {
		return FederatedPeerRow{}, fmt.Errorf("get federated peer: %w", err)
	}
	return row, nil
}

func (s *Store) ListFederatedPeers(ctx context.Context) ([]FederatedPeerRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT server_id, public_url, signing_public_key, status, trusted_at, created_at
		FROM federated_peers WHERE status = 'trusted'
		ORDER BY trusted_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list federated peers: %w", err)
	}
	defer rows.Close()
	var out []FederatedPeerRow
	for rows.Next() {
		var row FederatedPeerRow
		if err := rows.Scan(
			&row.ServerID, &row.PublicURL, &row.SigningPublicKey, &row.Status, &row.TrustedAt, &row.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) UpsertConversationPeer(
	ctx context.Context, channelID uuid.UUID, peerServerID, peerChannelID string,
) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO conversation_peers (channel_id, peer_server_id, peer_channel_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (channel_id, peer_server_id) DO UPDATE
		SET peer_channel_id = EXCLUDED.peer_channel_id
	`, channelID, peerServerID, peerChannelID)
	if err != nil {
		return fmt.Errorf("upsert conversation peer: %w", err)
	}
	return nil
}

func (s *Store) ListConversationPeers(ctx context.Context, channelID uuid.UUID) ([]ConversationPeerRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT channel_id, peer_server_id, peer_channel_id, created_at
		FROM conversation_peers WHERE channel_id = $1
	`, channelID)
	if err != nil {
		return nil, fmt.Errorf("list conversation peers: %w", err)
	}
	defer rows.Close()
	var out []ConversationPeerRow
	for rows.Next() {
		var row ConversationPeerRow
		if err := rows.Scan(&row.ChannelID, &row.PeerServerID, &row.PeerChannelID, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) FindChannelByPeer(
	ctx context.Context, peerServerID, peerChannelID string,
) (ChannelRow, error) {
	var channelID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT channel_id FROM conversation_peers
		WHERE peer_server_id = $1 AND peer_channel_id = $2
	`, peerServerID, peerChannelID).Scan(&channelID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChannelRow{}, ErrNotFound
	}
	if err != nil {
		return ChannelRow{}, fmt.Errorf("find channel by peer: %w", err)
	}
	return s.GetChannel(ctx, channelID)
}

func (s *Store) EnsureShadowUser(
	ctx context.Context,
	remoteServerID, remoteUserID, displayName string,
) (UserRow, error) {
	var row UserRow
	err := s.pool.QueryRow(ctx, `
		SELECT `+userSelectColumns+`
		FROM users
		WHERE is_remote = true AND remote_server_id = $1 AND remote_user_id = $2
	`, remoteServerID, remoteUserID).Scan(scanUserFields(&row)...)
	if err == nil {
		if displayName != "" && displayName != row.DisplayName {
			_, _ = s.pool.Exec(ctx, `UPDATE users SET display_name = $2, updated_at = now() WHERE id = $1`, row.ID, displayName)
			row.DisplayName = displayName
		}
		return row, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return UserRow{}, fmt.Errorf("load shadow user: %w", err)
	}
	email := fmt.Sprintf("remote+%s@%s.invalid", remoteUserID, remoteServerID)
	err = s.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name, is_remote, remote_server_id, remote_user_id, email_verified, email_verified_at)
		VALUES ($1, '', $2, true, $3, $4, true, now())
		RETURNING `+userSelectColumns+`
	`, email, displayName, remoteServerID, remoteUserID).Scan(scanUserFields(&row)...)
	if err != nil {
		return UserRow{}, fmt.Errorf("create shadow user: %w", err)
	}
	return row, nil
}

func (s *Store) UpsertFederatedMessageRef(
	ctx context.Context, channelID uuid.UUID, originServerID, originMessageID string, localMessageID uuid.UUID,
) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO federated_message_refs (channel_id, origin_server_id, origin_message_id, local_message_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (channel_id, origin_server_id, origin_message_id) DO NOTHING
	`, channelID, originServerID, originMessageID, localMessageID)
	if err != nil {
		return false, fmt.Errorf("upsert federated message ref: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) GetFederatedMessageRef(
	ctx context.Context, channelID uuid.UUID, originServerID, originMessageID string,
) (uuid.UUID, error) {
	var localID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT local_message_id FROM federated_message_refs
		WHERE channel_id = $1 AND origin_server_id = $2 AND origin_message_id = $3
	`, channelID, originServerID, originMessageID).Scan(&localID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get federated message ref: %w", err)
	}
	return localID, nil
}

type FederatedMessageOrigin struct {
	ChannelID       uuid.UUID
	OriginServerID  string
	OriginMessageID string
	LocalMessageID  uuid.UUID
}

func (s *Store) GetFederatedMessageRefByLocal(
	ctx context.Context, localMessageID uuid.UUID,
) (FederatedMessageOrigin, error) {
	var row FederatedMessageOrigin
	err := s.pool.QueryRow(ctx, `
		SELECT channel_id, origin_server_id, origin_message_id, local_message_id
		FROM federated_message_refs WHERE local_message_id = $1
	`, localMessageID).Scan(
		&row.ChannelID, &row.OriginServerID, &row.OriginMessageID, &row.LocalMessageID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return FederatedMessageOrigin{}, ErrNotFound
	}
	if err != nil {
		return FederatedMessageOrigin{}, fmt.Errorf("get federated message ref by local: %w", err)
	}
	return row, nil
}

func (s *Store) ClaimFederatedEvent(
	ctx context.Context, originServerID, originEventID, kind string,
) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO federated_event_refs (origin_server_id, origin_event_id, kind)
		VALUES ($1, $2, $3)
		ON CONFLICT (origin_server_id, origin_event_id) DO NOTHING
	`, originServerID, originEventID, kind)
	if err != nil {
		return false, fmt.Errorf("claim federated event: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) RevokeFederatedPeer(ctx context.Context, serverID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE federated_peers SET status = 'revoked' WHERE server_id = $1
	`, serverID)
	if err != nil {
		return fmt.Errorf("revoke federated peer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteConversationPeers(ctx context.Context, channelID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM conversation_peers WHERE channel_id = $1`, channelID)
	if err != nil {
		return fmt.Errorf("delete conversation peers: %w", err)
	}
	return nil
}

func (s *Store) DeleteConversationPeerByRemote(
	ctx context.Context, peerServerID, peerChannelID string,
) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM conversation_peers
		WHERE peer_server_id = $1 AND peer_channel_id = $2
	`, peerServerID, peerChannelID)
	if err != nil {
		return fmt.Errorf("delete conversation peer by remote: %w", err)
	}
	return nil
}

func (s *Store) MarkChannelShared(ctx context.Context, channelID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE channels SET sharing_mode = 'shared', updated_at = now() WHERE id = $1
	`, channelID)
	if err != nil {
		return fmt.Errorf("mark channel shared: %w", err)
	}
	return nil
}

func (s *Store) MarkChannelLocal(ctx context.Context, channelID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE channels SET sharing_mode = 'local', updated_at = now() WHERE id = $1
	`, channelID)
	if err != nil {
		return fmt.Errorf("mark channel local: %w", err)
	}
	return nil
}
