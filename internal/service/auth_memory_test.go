package service_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/auth"
	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/id"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type memStore struct {
	mu                 sync.Mutex
	users              map[string]postgres.UserRow
	byID               map[uuid.UUID]postgres.UserRow
	sessions           map[string]postgres.SessionRow
	avatars            map[uuid.UUID]postgres.UserAvatar
	verificationTokens map[string]postgres.EmailVerificationTokenRow
}

func newMemStore() *memStore {
	return &memStore{
		users:              map[string]postgres.UserRow{},
		byID:               map[uuid.UUID]postgres.UserRow{},
		sessions:           map[string]postgres.SessionRow{},
		avatars:            map[uuid.UUID]postgres.UserAvatar{},
		verificationTokens: map[string]postgres.EmailVerificationTokenRow{},
	}
}

func (m *memStore) CreateUser(_ context.Context, email, displayName, passwordHash string) (postgres.UserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[email]; ok {
		return postgres.UserRow{}, postgres.ErrEmailConflict
	}
	now := time.Now().UTC()
	row := postgres.UserRow{
		ID:                id.New(),
		Email:             email,
		DisplayName:       displayName,
		PasswordHash:      passwordHash,
		NotificationLevel: string(domain.NotifyMentions),
		EmailVerified:     false,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	m.users[email] = row
	m.byID[row.ID] = row
	return row, nil
}

func (m *memStore) GetUserByEmail(_ context.Context, email string) (postgres.UserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.users[email]
	if !ok {
		return postgres.UserRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *memStore) GetUserByID(_ context.Context, id uuid.UUID) (postgres.UserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.byID[id]
	if !ok {
		return postgres.UserRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *memStore) UpdateUserProfile(
	_ context.Context, userID uuid.UUID, displayName string, notificationLevel domain.NotificationLevel,
) (postgres.UserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.byID[userID]
	if !ok {
		return postgres.UserRow{}, postgres.ErrNotFound
	}
	row.DisplayName = displayName
	row.NotificationLevel = string(notificationLevel)
	row.UpdatedAt = time.Now().UTC()
	m.byID[userID] = row
	m.users[row.Email] = row
	return row, nil
}

func (m *memStore) UpdateUserStatus(
	_ context.Context, userID uuid.UUID, statusEmoji, statusText string, statusExpiresAt *time.Time,
) (postgres.UserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.byID[userID]
	if !ok {
		return postgres.UserRow{}, postgres.ErrNotFound
	}
	row.StatusEmoji = statusEmoji
	row.StatusText = statusText
	row.StatusExpiresAt = statusExpiresAt
	row.UpdatedAt = time.Now().UTC()
	m.byID[userID] = row
	m.users[row.Email] = row
	return row, nil
}

func (m *memStore) SetUserAvatar(
	_ context.Context, userID uuid.UUID, contentType string, data []byte,
) (postgres.UserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.byID[userID]
	if !ok {
		return postgres.UserRow{}, postgres.ErrNotFound
	}
	now := time.Now().UTC()
	m.avatars[userID] = postgres.UserAvatar{
		ContentType: contentType, Bytes: append([]byte(nil), data...), UpdatedAt: now,
	}
	row.HasAvatar = true
	row.AvatarContentType = contentType
	row.AvatarUpdatedAt = &now
	row.UpdatedAt = now
	m.byID[userID] = row
	m.users[row.Email] = row
	return row, nil
}

func (m *memStore) ClearUserAvatar(_ context.Context, userID uuid.UUID) (postgres.UserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.byID[userID]
	if !ok {
		return postgres.UserRow{}, postgres.ErrNotFound
	}
	delete(m.avatars, userID)
	row.HasAvatar = false
	row.AvatarContentType = ""
	row.AvatarUpdatedAt = nil
	row.UpdatedAt = time.Now().UTC()
	m.byID[userID] = row
	m.users[row.Email] = row
	return row, nil
}

func (m *memStore) GetUserAvatar(_ context.Context, userID uuid.UUID) (postgres.UserAvatar, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	avatar, ok := m.avatars[userID]
	if !ok {
		return postgres.UserAvatar{}, postgres.ErrNotFound
	}
	return avatar, nil
}

func (m *memStore) CreateSession(_ context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (postgres.SessionRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row := postgres.SessionRow{
		ID:        id.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}
	m.sessions[tokenHash] = row
	return row, nil
}

func (m *memStore) GetSessionByTokenHash(_ context.Context, tokenHash string) (postgres.SessionRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.sessions[tokenHash]
	if !ok {
		return postgres.SessionRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *memStore) DeleteSessionByTokenHash(_ context.Context, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, tokenHash)
	return nil
}

func (m *memStore) CreateEmailVerificationToken(
	_ context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time,
) (postgres.EmailVerificationTokenRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row := postgres.EmailVerificationTokenRow{
		ID: id.New(), UserID: userID, TokenHash: tokenHash,
		ExpiresAt: expiresAt, CreatedAt: time.Now().UTC(),
	}
	m.verificationTokens[tokenHash] = row
	return row, nil
}

func (m *memStore) DeleteEmailVerificationTokensForUser(_ context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for hash, row := range m.verificationTokens {
		if row.UserID == userID {
			delete(m.verificationTokens, hash)
		}
	}
	return nil
}

func (m *memStore) LatestEmailVerificationTokenCreatedAt(
	_ context.Context, userID uuid.UUID,
) (*time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var latest *time.Time
	for _, row := range m.verificationTokens {
		if row.UserID != userID {
			continue
		}
		created := row.CreatedAt
		if latest == nil || created.After(*latest) {
			latest = &created
		}
	}
	return latest, nil
}

func (m *memStore) GetEmailVerificationTokenByHash(
	_ context.Context, tokenHash string,
) (postgres.EmailVerificationTokenRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.verificationTokens[tokenHash]
	if !ok {
		return postgres.EmailVerificationTokenRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *memStore) MarkUserEmailVerified(
	_ context.Context, userID uuid.UUID, verifiedAt time.Time,
) (postgres.UserRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.byID[userID]
	if !ok {
		return postgres.UserRow{}, postgres.ErrNotFound
	}
	row.EmailVerified = true
	row.EmailVerifiedAt = &verifiedAt
	row.UpdatedAt = time.Now().UTC()
	m.byID[userID] = row
	m.users[row.Email] = row
	return row, nil
}

func testPolicy() auth.PasswordPolicy {
	return auth.PasswordPolicy{
		MinLength:        12,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireNumber:    true,
		RequireSymbol:    false,
	}
}

func TestAuthSignupLoginMeLogout(t *testing.T) {
	t.Parallel()

	svc := service.NewAuthService(newMemStore(), testPolicy(), time.Hour)
	ctx := context.Background()

	signup, err := svc.Signup(ctx, service.SignupInput{
		Email:       "Avery@Example.com",
		Password:    "ValidPass1234",
		DisplayName: "Avery Chen",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	if signup.Token == "" || signup.User.Email != "avery@example.com" {
		t.Fatalf("unexpected signup result: %+v", signup)
	}

	if _, err := svc.Signup(ctx, service.SignupInput{
		Email:       "avery@example.com",
		Password:    "ValidPass1234",
		DisplayName: "Avery",
	}); !errors.Is(err, service.ErrEmailTaken) {
		t.Fatalf("expected email taken, got %v", err)
	}

	me, err := svc.Me(ctx, signup.Token)
	if err != nil || me.DisplayName != "Avery Chen" {
		t.Fatalf("me: %#v err=%v", me, err)
	}

	login, err := svc.Login(ctx, service.LoginInput{
		Email:    "avery@example.com",
		Password: "ValidPass1234",
	})
	if err != nil || login.Token == "" {
		t.Fatalf("login: %#v err=%v", login, err)
	}

	if _, err := svc.Login(ctx, service.LoginInput{
		Email:    "avery@example.com",
		Password: "WrongPass1234",
	}); !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}

	if err := svc.Logout(ctx, login.Token); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Me(ctx, login.Token); !errors.Is(err, service.ErrUnauthorized) {
		t.Fatalf("expected unauthorized after logout, got %v", err)
	}
}

func TestAuthWeakPassword(t *testing.T) {
	t.Parallel()
	svc := service.NewAuthService(newMemStore(), testPolicy(), time.Hour)
	_, err := svc.Signup(context.Background(), service.SignupInput{
		Email:       "weak@example.com",
		Password:    "short",
		DisplayName: "Weak",
	})
	if !errors.Is(err, service.ErrWeakPassword) {
		t.Fatalf("expected weak password error, got %v", err)
	}
}

func TestAuthUpdateProfile(t *testing.T) {
	t.Parallel()

	svc := service.NewAuthService(newMemStore(), testPolicy(), time.Hour)
	ctx := context.Background()

	signup, err := svc.Signup(ctx, service.SignupInput{
		Email:       "profile@example.com",
		Password:    "ValidPass1234",
		DisplayName: "Before",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	updated, err := svc.UpdateProfile(ctx, signup.User.ID, "  After Name  ", "nothing")
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.DisplayName != "After Name" {
		t.Fatalf("display name = %q", updated.DisplayName)
	}
	if updated.NotificationLevel != domain.NotifyNothing {
		t.Fatalf("notification level = %q", updated.NotificationLevel)
	}

	me, err := svc.Me(ctx, signup.Token)
	if err != nil || me.DisplayName != "After Name" || me.NotificationLevel != domain.NotifyNothing {
		t.Fatalf("me after update: %#v err=%v", me, err)
	}

	if _, err := svc.UpdateProfile(ctx, signup.User.ID, "   ", "mentions"); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("expected invalid input for empty name, got %v", err)
	}
	if _, err := svc.UpdateProfile(ctx, signup.User.ID, "Name", "loud"); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("expected invalid input for bad level, got %v", err)
	}
}

type capturingVerificationMailer struct {
	to        string
	verifyURL string
	calls     int
}

func (m *capturingVerificationMailer) EnqueueVerificationEmail(_ context.Context, to, verifyURL string) error {
	m.calls++
	m.to = to
	m.verifyURL = verifyURL
	return nil
}

func tokenFromVerifyURL(verifyURL string) string {
	_, token, ok := strings.Cut(verifyURL, "token=")
	if !ok {
		return ""
	}
	return token
}

func TestAuthEmailVerification(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	mailer := &capturingVerificationMailer{}
	svc := service.NewAuthService(store, testPolicy(), time.Hour).
		WithVerificationMailer(mailer, "http://localhost:5173")
	ctx := context.Background()

	signup, err := svc.Signup(ctx, service.SignupInput{
		Email:       "verify@example.com",
		Password:    "ValidPass1234",
		DisplayName: "Verify Me",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	if signup.User.EmailVerified {
		t.Fatal("expected unverified signup")
	}
	if mailer.calls != 1 || mailer.to != "verify@example.com" {
		t.Fatalf("expected verification email, got calls=%d to=%q", mailer.calls, mailer.to)
	}
	rawToken := tokenFromVerifyURL(mailer.verifyURL)
	if rawToken == "" {
		t.Fatalf("missing token in verify URL %q", mailer.verifyURL)
	}

	if _, err := svc.VerifyEmail(ctx, "not-a-token"); !errors.Is(err, service.ErrInvalidVerificationToken) {
		t.Fatalf("expected invalid token, got %v", err)
	}

	verified, err := svc.VerifyEmail(ctx, rawToken)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !verified.EmailVerified || verified.EmailVerifiedAt == nil {
		t.Fatalf("expected verified user, got %#v", verified)
	}

	if _, err := svc.VerifyEmail(ctx, rawToken); !errors.Is(err, service.ErrInvalidVerificationToken) {
		t.Fatalf("expected reused token to fail, got %v", err)
	}

	if err := svc.ResendVerificationEmail(ctx, signup.User.ID); err != nil {
		t.Fatalf("resend when already verified: %v", err)
	}
	if mailer.calls != 1 {
		t.Fatalf("expected no extra email when already verified, calls=%d", mailer.calls)
	}
}

func TestAuthResendVerificationRateLimit(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	mailer := &capturingVerificationMailer{}
	svc := service.NewAuthService(store, testPolicy(), time.Hour).
		WithVerificationMailer(mailer, "http://localhost:5173")
	ctx := context.Background()

	signup, err := svc.Signup(ctx, service.SignupInput{
		Email:       "rate@example.com",
		Password:    "ValidPass1234",
		DisplayName: "Rate Limit",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	if err := svc.ResendVerificationEmail(ctx, signup.User.ID); !errors.Is(err, service.ErrVerificationRateLimited) {
		t.Fatalf("expected rate limit, got %v", err)
	}
	if mailer.calls != 1 {
		t.Fatalf("expected single email, calls=%d", mailer.calls)
	}
}

func TestAuthAvatar(t *testing.T) {
	t.Parallel()

	svc := service.NewAuthService(newMemStore(), testPolicy(), time.Hour)
	ctx := context.Background()

	signup, err := svc.Signup(ctx, service.SignupInput{
		Email:       "avatar@example.com",
		Password:    "ValidPass1234",
		DisplayName: "Avatar User",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	}
	updated, err := svc.SetAvatar(ctx, signup.User.ID, bytes.NewReader(png), "image/png")
	if err != nil {
		t.Fatalf("set avatar: %v", err)
	}
	if !updated.HasAvatar || updated.AvatarUpdatedAt == nil {
		t.Fatalf("expected avatar metadata, got %#v", updated)
	}

	avatar, err := svc.GetAvatar(ctx, signup.User.ID)
	if err != nil || avatar.ContentType != "image/png" || len(avatar.Bytes) == 0 {
		t.Fatalf("get avatar: %#v err=%v", avatar, err)
	}

	cleared, err := svc.ClearAvatar(ctx, signup.User.ID)
	if err != nil || cleared.HasAvatar {
		t.Fatalf("clear avatar: %#v err=%v", cleared, err)
	}
	if _, err := svc.GetAvatar(ctx, signup.User.ID); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("expected not found after clear, got %v", err)
	}
}
