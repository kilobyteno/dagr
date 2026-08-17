-- name: CreateUser :one
INSERT INTO users (email, display_name, password_hash)
VALUES ($1, $2, $3)
RETURNING id, email, display_name, password_hash, notification_level,
    (avatar_bytes IS NOT NULL), COALESCE(avatar_content_type, ''), avatar_updated_at,
    created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, display_name, password_hash, notification_level,
    (avatar_bytes IS NOT NULL), COALESCE(avatar_content_type, ''), avatar_updated_at,
    created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, display_name, password_hash, notification_level,
    (avatar_bytes IS NOT NULL), COALESCE(avatar_content_type, ''), avatar_updated_at,
    created_at, updated_at
FROM users
WHERE id = $1;

-- name: UpdateUserProfile :one
UPDATE users
SET display_name = $2,
    notification_level = $3,
    updated_at = now()
WHERE id = $1
RETURNING id, email, display_name, password_hash, notification_level,
    (avatar_bytes IS NOT NULL), COALESCE(avatar_content_type, ''), avatar_updated_at,
    created_at, updated_at;

-- name: SetUserAvatar :one
UPDATE users
SET avatar_bytes = $2,
    avatar_content_type = $3,
    avatar_updated_at = now(),
    updated_at = now()
WHERE id = $1
RETURNING id, email, display_name, password_hash, notification_level,
    (avatar_bytes IS NOT NULL), COALESCE(avatar_content_type, ''), avatar_updated_at,
    created_at, updated_at;

-- name: ClearUserAvatar :one
UPDATE users
SET avatar_bytes = NULL,
    avatar_content_type = NULL,
    avatar_updated_at = NULL,
    updated_at = now()
WHERE id = $1
RETURNING id, email, display_name, password_hash, notification_level,
    (avatar_bytes IS NOT NULL), COALESCE(avatar_content_type, ''), avatar_updated_at,
    created_at, updated_at;

-- name: GetUserAvatar :one
SELECT avatar_bytes, COALESCE(avatar_content_type, ''), avatar_updated_at
FROM users
WHERE id = $1 AND avatar_bytes IS NOT NULL;

-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, created_at;

-- name: GetSessionByTokenHash :one
SELECT id, user_id, token_hash, expires_at, created_at
FROM sessions
WHERE token_hash = $1;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions
WHERE token_hash = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at <= now();
