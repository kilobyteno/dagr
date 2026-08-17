package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/auth"
	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
)

var (
	ErrInvalidCredentials       = errors.New("invalid email or password")
	ErrInvalidInput             = errors.New("invalid input")
	ErrUnauthorized             = errors.New("unauthorized")
	ErrEmailTaken               = errors.New("email already registered")
	ErrWeakPassword             = errors.New("password does not meet policy")
	ErrInvalidAvatar            = errors.New("invalid profile picture")
	ErrInvalidVerificationToken = errors.New("invalid or expired verification token")
	ErrVerificationRateLimited  = errors.New("verification email rate limited")
)

const (
	emailVerificationTTL       = 24 * time.Hour
	emailVerificationResendMin = 60 * time.Second
)

const maxUserAvatarBytes = 2 << 20 // 2 MiB

var allowedUserAvatarTypes = map[string]string{
	"image/png":  "image/png",
	"image/jpeg": "image/jpeg",
	"image/jpg":  "image/jpeg",
	"image/webp": "image/webp",
	"image/gif":  "image/gif",
}

type AuthStore interface {
	CreateUser(ctx context.Context, email, displayName, passwordHash string) (postgres.UserRow, error)
	GetUserByEmail(ctx context.Context, email string) (postgres.UserRow, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (postgres.UserRow, error)
	UpdateUserProfile(ctx context.Context, id uuid.UUID, displayName string, notificationLevel domain.NotificationLevel) (postgres.UserRow, error)
	UpdateUserStatus(ctx context.Context, id uuid.UUID, statusEmoji, statusText string, statusExpiresAt *time.Time) (postgres.UserRow, error)
	SetUserAvatar(ctx context.Context, userID uuid.UUID, contentType string, data []byte) (postgres.UserRow, error)
	ClearUserAvatar(ctx context.Context, userID uuid.UUID) (postgres.UserRow, error)
	GetUserAvatar(ctx context.Context, userID uuid.UUID) (postgres.UserAvatar, error)
	CreateSession(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (postgres.SessionRow, error)
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (postgres.SessionRow, error)
	DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error
	CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (postgres.EmailVerificationTokenRow, error)
	DeleteEmailVerificationTokensForUser(ctx context.Context, userID uuid.UUID) error
	LatestEmailVerificationTokenCreatedAt(ctx context.Context, userID uuid.UUID) (*time.Time, error)
	GetEmailVerificationTokenByHash(ctx context.Context, tokenHash string) (postgres.EmailVerificationTokenRow, error)
	MarkUserEmailVerified(ctx context.Context, userID uuid.UUID, verifiedAt time.Time) (postgres.UserRow, error)
}

// SignupAutoJoiner adds a new user to workspaces matching their email domain.
type SignupAutoJoiner interface {
	AutoJoinForEmail(ctx context.Context, userID uuid.UUID, email string) error
}

// VerificationMailer enqueues account email verification messages.
type VerificationMailer interface {
	EnqueueVerificationEmail(ctx context.Context, to, verifyURL string) error
}

type noopVerificationMailer struct{}

func (noopVerificationMailer) EnqueueVerificationEmail(context.Context, string, string) error {
	return nil
}

type AuthService struct {
	store      AuthStore
	policy     auth.PasswordPolicy
	sessionTTL time.Duration
	autoJoin   SignupAutoJoiner
	mailer     VerificationMailer
	baseURL    string
}

func NewAuthService(store AuthStore, policy auth.PasswordPolicy, sessionTTL time.Duration) *AuthService {
	if sessionTTL <= 0 {
		sessionTTL = 30 * 24 * time.Hour
	}
	return &AuthService{
		store:      store,
		policy:     policy,
		sessionTTL: sessionTTL,
		mailer:     noopVerificationMailer{},
		baseURL:    "http://localhost:5173",
	}
}

// WithAutoJoiner configures optional workspace auto-join after signup.
func (s *AuthService) WithAutoJoiner(joiner SignupAutoJoiner) *AuthService {
	s.autoJoin = joiner
	return s
}

// WithVerificationMailer configures verification email delivery and public base URL for links.
func (s *AuthService) WithVerificationMailer(mailer VerificationMailer, baseURL string) *AuthService {
	if mailer == nil {
		mailer = noopVerificationMailer{}
	}
	s.mailer = mailer
	if strings.TrimSpace(baseURL) != "" {
		s.baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	}
	return s
}

type AuthResult struct {
	Token     string
	ExpiresAt time.Time
	User      domain.User
}

type SignupInput struct {
	Email       string
	Password    string
	DisplayName string
}

type LoginInput struct {
	Email    string
	Password string
}

func (s *AuthService) Signup(ctx context.Context, in SignupInput) (*AuthResult, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	displayName := strings.TrimSpace(in.DisplayName)
	if email == "" || displayName == "" || in.Password == "" {
		return nil, ErrInvalidInput
	}
	if err := s.policy.Validate(in.Password); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrWeakPassword, err.Error())
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.store.CreateUser(ctx, email, displayName, hash)
	if err != nil {
		if errors.Is(err, postgres.ErrEmailConflict) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	if s.autoJoin != nil {
		if err := s.autoJoin.AutoJoinForEmail(ctx, user.ID, email); err != nil {
			return nil, err
		}
	}

	_ = s.issueVerificationEmail(ctx, user, false)

	return s.issueSession(ctx, user)
}

// VerifyEmail marks the account verified when the token is valid.
func (s *AuthService) VerifyEmail(ctx context.Context, rawToken string) (*domain.User, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, ErrInvalidVerificationToken
	}
	tokenHash := auth.HashToken(rawToken)
	row, err := s.store.GetEmailVerificationTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrInvalidVerificationToken
		}
		return nil, err
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		_ = s.store.DeleteEmailVerificationTokensForUser(ctx, row.UserID)
		return nil, ErrInvalidVerificationToken
	}
	user, err := s.store.MarkUserEmailVerified(ctx, row.UserID, time.Now().UTC())
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrInvalidVerificationToken
		}
		return nil, err
	}
	_ = s.store.DeleteEmailVerificationTokensForUser(ctx, row.UserID)
	out := user.ToDomain()
	return &out, nil
}

// ResendVerificationEmail issues a new verification link for the authenticated user.
func (s *AuthService) ResendVerificationEmail(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidInput
	}
	user, err := s.store.GetUserByID(ctx, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrUnauthorized
		}
		return err
	}
	if user.EmailVerified {
		return nil
	}
	return s.issueVerificationEmail(ctx, user, true)
}

func (s *AuthService) issueVerificationEmail(ctx context.Context, user postgres.UserRow, rateLimit bool) error {
	if user.EmailVerified {
		return nil
	}
	if rateLimit {
		latest, err := s.store.LatestEmailVerificationTokenCreatedAt(ctx, user.ID)
		if err != nil {
			return err
		}
		if latest != nil && time.Since(latest.UTC()) < emailVerificationResendMin {
			return ErrVerificationRateLimited
		}
	}
	rawToken, tokenHash, err := auth.NewSessionToken(auth.DefaultTokenBytes)
	if err != nil {
		return err
	}
	_ = s.store.DeleteEmailVerificationTokensForUser(ctx, user.ID)
	expiresAt := time.Now().UTC().Add(emailVerificationTTL)
	if _, err := s.store.CreateEmailVerificationToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return err
	}
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.baseURL, rawToken)
	return s.mailer.EnqueueVerificationEmail(ctx, user.Email, verifyURL)
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (*AuthResult, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	if email == "" || in.Password == "" {
		return nil, ErrInvalidInput
	}

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	ok, err := auth.VerifyPassword(user.PasswordHash, in.Password)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	return s.issueSession(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	if strings.TrimSpace(rawToken) == "" {
		return ErrUnauthorized
	}
	return s.store.DeleteSessionByTokenHash(ctx, auth.HashToken(rawToken))
}

func (s *AuthService) Me(ctx context.Context, rawToken string) (*domain.User, error) {
	user, _, err := s.Authenticate(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateProfile updates the authenticated user's display name and notification preference.
func (s *AuthService) UpdateProfile(
	ctx context.Context, userID, displayName, notificationLevel string,
) (*domain.User, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len(displayName) > 80 {
		return nil, ErrInvalidInput
	}
	level, ok := domain.ParseNotificationLevel(strings.TrimSpace(notificationLevel))
	if !ok {
		return nil, ErrInvalidInput
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	row, err := s.store.UpdateUserProfile(ctx, uid, displayName, level)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	out := row.ToDomain()
	return &out, nil
}

const (
	maxStatusEmojiRunes = 32
	maxStatusTextRunes  = 100
	maxStatusDuration   = 30 * 24 * time.Hour
)

// UpdateStatus sets or clears the user's custom status emoji and text.
func (s *AuthService) UpdateStatus(
	ctx context.Context, userID, statusEmoji, statusText string, expiresAt *time.Time,
) (*domain.User, error) {
	statusEmoji = strings.TrimSpace(statusEmoji)
	statusText = strings.Join(strings.Fields(strings.TrimSpace(statusText)), " ")
	if len([]rune(statusEmoji)) > maxStatusEmojiRunes {
		return nil, ErrInvalidInput
	}
	if len([]rune(statusText)) > maxStatusTextRunes {
		return nil, ErrInvalidInput
	}
	if statusEmoji == "" && statusText == "" {
		expiresAt = nil
	} else if expiresAt != nil {
		expires := expiresAt.UTC()
		now := time.Now().UTC()
		if !expires.After(now) || expires.Sub(now) > maxStatusDuration {
			return nil, ErrInvalidInput
		}
		expiresAt = &expires
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	row, err := s.store.UpdateUserStatus(ctx, uid, statusEmoji, statusText, expiresAt)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	out := row.ToDomain()
	return &out, nil
}

func (s *AuthService) SetAvatar(
	ctx context.Context,
	userID string,
	r io.Reader,
	declaredType string,
) (*domain.User, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	limited := io.LimitReader(r, maxUserAvatarBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > maxUserAvatarBytes {
		return nil, ErrInvalidAvatar
	}
	detected := http.DetectContentType(data)
	contentType, ok := allowedUserAvatarTypes[detected]
	if !ok {
		if declared := strings.ToLower(strings.TrimSpace(declaredType)); declared != "" {
			contentType, ok = allowedUserAvatarTypes[declared]
		}
	}
	if !ok {
		return nil, ErrInvalidAvatar
	}
	row, err := s.store.SetUserAvatar(ctx, uid, contentType, data)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	out := row.ToDomain()
	return &out, nil
}

func (s *AuthService) ClearAvatar(ctx context.Context, userID string) (*domain.User, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	row, err := s.store.ClearUserAvatar(ctx, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	out := row.ToDomain()
	return &out, nil
}

func (s *AuthService) GetAvatar(ctx context.Context, userID string) (*postgres.UserAvatar, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrNotFound
	}
	avatar, err := s.store.GetUserAvatar(ctx, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &avatar, nil
}

func (s *AuthService) Authenticate(ctx context.Context, rawToken string) (*domain.User, string, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, "", ErrUnauthorized
	}

	tokenHash := auth.HashToken(rawToken)
	session, err := s.store.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, "", ErrUnauthorized
		}
		return nil, "", err
	}
	if time.Now().After(session.ExpiresAt) {
		_ = s.store.DeleteSessionByTokenHash(ctx, tokenHash)
		return nil, "", ErrUnauthorized
	}

	user, err := s.store.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, "", ErrUnauthorized
		}
		return nil, "", err
	}

	domainUser := user.ToDomain()
	return &domainUser, tokenHash, nil
}

func (s *AuthService) issueSession(ctx context.Context, user postgres.UserRow) (*AuthResult, error) {
	rawToken, tokenHash, err := auth.NewSessionToken(auth.DefaultTokenBytes)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(s.sessionTTL)
	if _, err := s.store.CreateSession(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return nil, err
	}
	return &AuthResult{
		Token:     rawToken,
		ExpiresAt: expiresAt,
		User:      user.ToDomain(),
	}, nil
}
