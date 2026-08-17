package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kilobyteno/dagr-chat/internal/domain"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrEmailConflict = errors.New("email already registered")
)

// Store is a Postgres-backed repository for auth entities.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type UserRow struct {
	ID                uuid.UUID
	Email             string
	DisplayName       string
	PasswordHash      string
	NotificationLevel string
	EmailVerified     bool
	EmailVerifiedAt   *time.Time
	StatusEmoji       string
	StatusText        string
	StatusExpiresAt   *time.Time
	HasAvatar         bool
	AvatarContentType string
	AvatarUpdatedAt   *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type UserAvatar struct {
	ContentType string
	Bytes       []byte
	UpdatedAt   time.Time
}

func (u UserRow) ToDomain() domain.User {
	level, ok := domain.ParseNotificationLevel(u.NotificationLevel)
	if !ok {
		level = domain.NotifyMentions
	}
	emoji, text, expiresAt := domain.EffectiveCustomStatus(u.StatusEmoji, u.StatusText, u.StatusExpiresAt)
	return domain.User{
		ID:                u.ID.String(),
		Email:             u.Email,
		DisplayName:       u.DisplayName,
		NotificationLevel: level,
		EmailVerified:     u.EmailVerified,
		EmailVerifiedAt:   u.EmailVerifiedAt,
		StatusEmoji:       emoji,
		StatusText:        text,
		StatusExpiresAt:   expiresAt,
		HasAvatar:         u.HasAvatar,
		AvatarUpdatedAt:   u.AvatarUpdatedAt,
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
	}
}

const userSelectColumns = `
	id, email, display_name, password_hash, notification_level,
	email_verified, email_verified_at,
	COALESCE(status_emoji, ''), COALESCE(status_text, ''), status_expires_at,
	(avatar_bytes IS NOT NULL), COALESCE(avatar_content_type, ''), avatar_updated_at,
	created_at, updated_at
`

func scanUserFields(row *UserRow) []any {
	return []any{
		&row.ID, &row.Email, &row.DisplayName, &row.PasswordHash, &row.NotificationLevel,
		&row.EmailVerified, &row.EmailVerifiedAt,
		&row.StatusEmoji, &row.StatusText, &row.StatusExpiresAt,
		&row.HasAvatar, &row.AvatarContentType, &row.AvatarUpdatedAt,
		&row.CreatedAt, &row.UpdatedAt,
	}
}

type SessionRow struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

func scanUserRow(row *UserRow, err error) (UserRow, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return UserRow{}, ErrNotFound
	}
	if err != nil {
		return UserRow{}, err
	}
	if row.NotificationLevel == "" {
		row.NotificationLevel = string(domain.NotifyMentions)
	}
	return *row, nil
}

func (s *Store) CreateUser(ctx context.Context, email, displayName, passwordHash string) (UserRow, error) {
	var row UserRow
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (email, display_name, password_hash, email_verified)
		VALUES ($1, $2, $3, false)
		RETURNING `+userSelectColumns+`
	`, email, displayName, passwordHash).Scan(scanUserFields(&row)...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return UserRow{}, ErrEmailConflict
		}
		return UserRow{}, fmt.Errorf("create user: %w", err)
	}
	return scanUserRow(&row, nil)
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (UserRow, error) {
	var row UserRow
	err := s.pool.QueryRow(ctx, `
		SELECT `+userSelectColumns+`
		FROM users
		WHERE email = $1
	`, email).Scan(scanUserFields(&row)...)
	if row, scanErr := scanUserRow(&row, err); scanErr != nil {
		if errors.Is(scanErr, ErrNotFound) {
			return UserRow{}, ErrNotFound
		}
		return UserRow{}, fmt.Errorf("get user by email: %w", scanErr)
	} else {
		return row, nil
	}
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (UserRow, error) {
	var row UserRow
	err := s.pool.QueryRow(ctx, `
		SELECT `+userSelectColumns+`
		FROM users
		WHERE id = $1
	`, id).Scan(scanUserFields(&row)...)
	if row, scanErr := scanUserRow(&row, err); scanErr != nil {
		if errors.Is(scanErr, ErrNotFound) {
			return UserRow{}, ErrNotFound
		}
		return UserRow{}, fmt.Errorf("get user by id: %w", scanErr)
	} else {
		return row, nil
	}
}

func (s *Store) UpdateUserProfile(
	ctx context.Context,
	id uuid.UUID,
	displayName string,
	notificationLevel domain.NotificationLevel,
) (UserRow, error) {
	var row UserRow
	err := s.pool.QueryRow(ctx, `
		UPDATE users
		SET display_name = $2,
			notification_level = $3,
			updated_at = now()
		WHERE id = $1
		RETURNING `+userSelectColumns+`
	`, id, displayName, string(notificationLevel)).Scan(scanUserFields(&row)...)
	if row, scanErr := scanUserRow(&row, err); scanErr != nil {
		if errors.Is(scanErr, ErrNotFound) {
			return UserRow{}, ErrNotFound
		}
		return UserRow{}, fmt.Errorf("update user profile: %w", scanErr)
	} else {
		return row, nil
	}
}

func (s *Store) UpdateUserStatus(
	ctx context.Context,
	id uuid.UUID,
	statusEmoji, statusText string,
	statusExpiresAt *time.Time,
) (UserRow, error) {
	var row UserRow
	err := s.pool.QueryRow(ctx, `
		UPDATE users
		SET status_emoji = $2,
			status_text = $3,
			status_expires_at = $4,
			updated_at = now()
		WHERE id = $1
		RETURNING `+userSelectColumns+`
	`, id, statusEmoji, statusText, statusExpiresAt).Scan(scanUserFields(&row)...)
	if row, scanErr := scanUserRow(&row, err); scanErr != nil {
		if errors.Is(scanErr, ErrNotFound) {
			return UserRow{}, ErrNotFound
		}
		return UserRow{}, fmt.Errorf("update user status: %w", scanErr)
	} else {
		return row, nil
	}
}

func (s *Store) SetUserAvatar(
	ctx context.Context,
	userID uuid.UUID,
	contentType string,
	data []byte,
) (UserRow, error) {
	var row UserRow
	err := s.pool.QueryRow(ctx, `
		UPDATE users
		SET avatar_bytes = $2,
			avatar_content_type = $3,
			avatar_updated_at = now(),
			updated_at = now()
		WHERE id = $1
		RETURNING `+userSelectColumns+`
	`, userID, data, contentType).Scan(scanUserFields(&row)...)
	if row, scanErr := scanUserRow(&row, err); scanErr != nil {
		if errors.Is(scanErr, ErrNotFound) {
			return UserRow{}, ErrNotFound
		}
		return UserRow{}, fmt.Errorf("set user avatar: %w", scanErr)
	} else {
		return row, nil
	}
}

func (s *Store) ClearUserAvatar(ctx context.Context, userID uuid.UUID) (UserRow, error) {
	var row UserRow
	err := s.pool.QueryRow(ctx, `
		UPDATE users
		SET avatar_bytes = NULL,
			avatar_content_type = NULL,
			avatar_updated_at = NULL,
			updated_at = now()
		WHERE id = $1
		RETURNING `+userSelectColumns+`
	`, userID).Scan(scanUserFields(&row)...)
	if row, scanErr := scanUserRow(&row, err); scanErr != nil {
		if errors.Is(scanErr, ErrNotFound) {
			return UserRow{}, ErrNotFound
		}
		return UserRow{}, fmt.Errorf("clear user avatar: %w", scanErr)
	} else {
		return row, nil
	}
}

func (s *Store) GetUserAvatar(ctx context.Context, userID uuid.UUID) (UserAvatar, error) {
	var avatar UserAvatar
	var updatedAt *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT avatar_bytes, COALESCE(avatar_content_type, ''), avatar_updated_at
		FROM users
		WHERE id = $1 AND avatar_bytes IS NOT NULL
	`, userID).Scan(&avatar.Bytes, &avatar.ContentType, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return UserAvatar{}, ErrNotFound
	}
	if err != nil {
		return UserAvatar{}, fmt.Errorf("get user avatar: %w", err)
	}
	if updatedAt != nil {
		avatar.UpdatedAt = *updatedAt
	}
	return avatar, nil
}

// UpdateUserDisplayName keeps older callers working; prefer UpdateUserProfile.
func (s *Store) UpdateUserDisplayName(ctx context.Context, id uuid.UUID, displayName string) (UserRow, error) {
	current, err := s.GetUserByID(ctx, id)
	if err != nil {
		return UserRow{}, err
	}
	level, ok := domain.ParseNotificationLevel(current.NotificationLevel)
	if !ok {
		level = domain.NotifyMentions
	}
	return s.UpdateUserProfile(ctx, id, displayName, level)
}

func (s *Store) ListUserNotificationLevels(
	ctx context.Context, userIDs []uuid.UUID,
) (map[uuid.UUID]domain.NotificationLevel, error) {
	out := map[uuid.UUID]domain.NotificationLevel{}
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, notification_level FROM users WHERE id = ANY($1)
	`, userIDs)
	if err != nil {
		return nil, fmt.Errorf("list user notification levels: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var level string
		if err := rows.Scan(&id, &level); err != nil {
			return nil, fmt.Errorf("scan user notification level: %w", err)
		}
		parsed, ok := domain.ParseNotificationLevel(level)
		if !ok {
			parsed = domain.NotifyMentions
		}
		out[id] = parsed
	}
	return out, rows.Err()
}

func (s *Store) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (SessionRow, error) {
	var row SessionRow
	err := s.pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, created_at
	`, userID, tokenHash, expiresAt).Scan(
		&row.ID, &row.UserID, &row.TokenHash, &row.ExpiresAt, &row.CreatedAt,
	)
	if err != nil {
		return SessionRow{}, fmt.Errorf("create session: %w", err)
	}
	return row, nil
}

func (s *Store) GetSessionByTokenHash(ctx context.Context, tokenHash string) (SessionRow, error) {
	var row SessionRow
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at
		FROM sessions
		WHERE token_hash = $1
	`, tokenHash).Scan(
		&row.ID, &row.UserID, &row.TokenHash, &row.ExpiresAt, &row.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionRow{}, ErrNotFound
	}
	if err != nil {
		return SessionRow{}, fmt.Errorf("get session: %w", err)
	}
	return row, nil
}

func (s *Store) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// EmailVerificationTokenRow is a stored email verification challenge.
type EmailVerificationTokenRow struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (s *Store) CreateEmailVerificationToken(
	ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time,
) (EmailVerificationTokenRow, error) {
	var row EmailVerificationTokenRow
	err := s.pool.QueryRow(ctx, `
		INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, created_at
	`, userID, tokenHash, expiresAt).Scan(
		&row.ID, &row.UserID, &row.TokenHash, &row.ExpiresAt, &row.CreatedAt,
	)
	if err != nil {
		return EmailVerificationTokenRow{}, fmt.Errorf("create email verification token: %w", err)
	}
	return row, nil
}

func (s *Store) DeleteEmailVerificationTokensForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM email_verification_tokens WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete email verification tokens: %w", err)
	}
	return nil
}

func (s *Store) LatestEmailVerificationTokenCreatedAt(
	ctx context.Context, userID uuid.UUID,
) (*time.Time, error) {
	var createdAt time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT created_at
		FROM email_verification_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("latest email verification token: %w", err)
	}
	return &createdAt, nil
}

func (s *Store) GetEmailVerificationTokenByHash(
	ctx context.Context, tokenHash string,
) (EmailVerificationTokenRow, error) {
	var row EmailVerificationTokenRow
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at
		FROM email_verification_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(
		&row.ID, &row.UserID, &row.TokenHash, &row.ExpiresAt, &row.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return EmailVerificationTokenRow{}, ErrNotFound
	}
	if err != nil {
		return EmailVerificationTokenRow{}, fmt.Errorf("get email verification token: %w", err)
	}
	return row, nil
}

func (s *Store) MarkUserEmailVerified(ctx context.Context, userID uuid.UUID, verifiedAt time.Time) (UserRow, error) {
	var row UserRow
	err := s.pool.QueryRow(ctx, `
		UPDATE users
		SET email_verified = true,
		    email_verified_at = $2,
		    updated_at = now()
		WHERE id = $1
		RETURNING `+userSelectColumns+`
	`, userID, verifiedAt).Scan(scanUserFields(&row)...)
	out, scanErr := scanUserRow(&row, err)
	if scanErr != nil {
		if errors.Is(scanErr, ErrNotFound) {
			return UserRow{}, ErrNotFound
		}
		return UserRow{}, fmt.Errorf("mark email verified: %w", scanErr)
	}
	return out, nil
}
