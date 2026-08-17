package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/auth"
	"github.com/kilobyteno/dagr-chat/internal/config"
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
		ID: id.New(), Email: email, DisplayName: displayName,
		PasswordHash: passwordHash, NotificationLevel: string(domain.NotifyMentions),
		EmailVerified: false,
		CreatedAt:     now, UpdatedAt: now,
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
		ID: id.New(), UserID: userID, TokenHash: tokenHash,
		ExpiresAt: expiresAt, CreatedAt: time.Now().UTC(),
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

type capturingVerificationMailer struct {
	to        string
	verifyURL string
}

func (m *capturingVerificationMailer) EnqueueVerificationEmail(_ context.Context, to, verifyURL string) error {
	m.to = to
	m.verifyURL = verifyURL
	return nil
}

func testServer() http.Handler {
	h, _, _ := testServerWithAuth()
	return h
}

func testServerWithAuth() (http.Handler, *memStore, *capturingVerificationMailer) {
	cfg := config.Config{
		PasswordPolicy: auth.PasswordPolicy{
			MinLength: 12, RequireUppercase: true, RequireLowercase: true, RequireNumber: true,
		},
		SessionTTL:    time.Hour,
		PublicBaseURL: "http://localhost:5173",
	}
	wsStore := newHTTPWorkspaceStore()
	authStore := newMemStore()
	wsStore.users = authStore
	mailer := &capturingVerificationMailer{}
	domainSvc := service.NewDomainService(wsStore)
	authSvc := service.NewAuthService(authStore, cfg.PasswordPolicy, cfg.SessionTTL).
		WithAutoJoiner(domainSvc).
		WithVerificationMailer(mailer, cfg.PublicBaseURL)
	workspaceSvc := service.NewWorkspaceService(wsStore)
	notificationSvc := service.NewNotificationService(wsStore)
	channelSvc := service.NewChannelService(wsStore).WithNotifications(notificationSvc)
	inviteSvc := service.NewInviteService(wsStore, cfg.PublicBaseURL, nil).
		WithNotifications(notificationSvc)
	messageSvc := service.NewMessageService(wsStore, channelSvc).
		WithNotifications(notificationSvc, notificationSvc)
	return NewRouter(
		cfg, authSvc, workspaceSvc, domainSvc,
		channelSvc, inviteSvc, messageSvc, notificationSvc,
		nil,
	), authStore, mailer
}

func TestHealth(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	testServer().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestPublicConfig(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/config", nil)
	rec := httptest.NewRecorder()
	testServer().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body publicConfigResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.PasswordPolicy.MinLength != 12 {
		t.Fatalf("minLength = %d", body.PasswordPolicy.MinLength)
	}
	if body.DeploymentMode != "selfhosted" {
		t.Fatalf("deploymentMode = %s", body.DeploymentMode)
	}
	if body.BillingEnabled {
		t.Fatal("self-hosted should not enable billing")
	}
}

func TestPublicConfigCloud(t *testing.T) {
	t.Parallel()
	cfg := config.Config{
		PasswordPolicy: auth.PasswordPolicy{
			MinLength: 12, RequireUppercase: true, RequireLowercase: true, RequireNumber: true,
		},
		DeploymentMode:        config.DeploymentCloud,
		BillingCurrency:       "EUR",
		ProMonthlyCents:       700,
		YearlyDiscountPercent: 10,
		EarlyAccessEnabled:    true,
		EarlyAccessMonths:     3,
		EarlyAccessPercentOff: 50,
	}
	h := NewRouter(cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/config", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body publicConfigResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !body.BillingEnabled || body.DeploymentMode != "cloud" {
		t.Fatalf("cloud config = %+v", body)
	}
	if body.Plans == nil || len(body.Plans.Plans) != 2 {
		t.Fatalf("plans = %+v", body.Plans)
	}
}

func TestSignupLoginMeLogout(t *testing.T) {
	t.Parallel()
	h := testServer()

	signupBody := []byte(`{"email":"casey@example.com","password":"ValidPass1234","displayName":"Casey"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(signupBody))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("signup status = %d body=%s", rec.Code, rec.Body.String())
	}
	var auth authResponse
	if err := json.NewDecoder(rec.Body).Decode(&auth); err != nil {
		t.Fatal(err)
	}
	if auth.Token == "" || auth.User.Email != "casey@example.com" {
		t.Fatalf("bad auth response: %+v", auth)
	}
	if auth.User.EmailVerified {
		t.Fatal("expected unverified email on signup")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("me status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d", rec.Code)
	}
}

func TestVerifyEmailAndResend(t *testing.T) {
	t.Parallel()
	h, _, mailer := testServerWithAuth()

	signupBody := []byte(`{"email":"verify-http@example.com","password":"ValidPass1234","displayName":"Verify"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(signupBody))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("signup status = %d body=%s", rec.Code, rec.Body.String())
	}
	var auth authResponse
	if err := json.NewDecoder(rec.Body).Decode(&auth); err != nil {
		t.Fatal(err)
	}
	if mailer.verifyURL == "" {
		t.Fatal("expected verification email to be enqueued")
	}
	rawToken := mailer.verifyURL
	if i := strings.Index(rawToken, "token="); i >= 0 {
		rawToken = rawToken[i+len("token="):]
	} else {
		t.Fatalf("token missing from %q", mailer.verifyURL)
	}

	verifyBody, _ := json.Marshal(map[string]string{"token": rawToken})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", bytes.NewReader(verifyBody))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify status = %d body=%s", rec.Code, rec.Body.String())
	}
	var meBody meResponse
	if err := json.NewDecoder(rec.Body).Decode(&meBody); err != nil {
		t.Fatal(err)
	}
	if !meBody.User.EmailVerified {
		t.Fatal("expected verified user")
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/me/email/resend-verification", nil)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("resend when verified status = %d", rec.Code)
	}
}

func TestUpdateProfile(t *testing.T) {
	t.Parallel()
	h := testServer()

	signupBody := []byte(`{"email":"profile@example.com","password":"ValidPass1234","displayName":"Before"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(signupBody))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("signup status = %d body=%s", rec.Code, rec.Body.String())
	}
	var auth authResponse
	if err := json.NewDecoder(rec.Body).Decode(&auth); err != nil {
		t.Fatal(err)
	}

	patchBody := []byte(`{"displayName":"After Name","notificationLevel":"all"}`)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/me", bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch me status = %d body=%s", rec.Code, rec.Body.String())
	}
	var me meResponse
	if err := json.NewDecoder(rec.Body).Decode(&me); err != nil {
		t.Fatal(err)
	}
	if me.User.DisplayName != "After Name" {
		t.Fatalf("display name = %q", me.User.DisplayName)
	}
	if me.User.NotificationLevel != "all" {
		t.Fatalf("notification level = %q", me.User.NotificationLevel)
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/me", bytes.NewReader([]byte(`{"displayName":"","notificationLevel":"mentions"}`)))
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty display name status = %d", rec.Code)
	}
}

func TestSignupDuplicateEmail(t *testing.T) {
	t.Parallel()
	h := testServer()
	body := []byte(`{"email":"dup@example.com","password":"ValidPass1234","displayName":"One"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first signup = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate signup = %d", rec.Code)
	}
}

type httpWorkspaceStore struct {
	mu             sync.Mutex
	workspaces     map[uuid.UUID]postgres.WorkspaceRow
	members        map[uuid.UUID]map[uuid.UUID]string
	handles        map[uuid.UUID]map[uuid.UUID]string
	aliases        map[uuid.UUID]map[string]uuid.UUID
	icons          map[uuid.UUID]postgres.WorkspaceIcon
	channels       map[uuid.UUID][]postgres.ChannelRow
	channelsByID   map[uuid.UUID]postgres.ChannelRow
	channelMembers map[uuid.UUID]map[uuid.UUID]string
	dmPairs        map[string]uuid.UUID
	slugs          map[string]uuid.UUID
	domains        map[uuid.UUID]postgres.WorkspaceDomainRow
	invites        map[uuid.UUID]postgres.WorkspaceInviteRow
	invitesByToken map[string]postgres.WorkspaceInviteRow
	messages       map[uuid.UUID]postgres.MessageRow
	messagesByCh   map[uuid.UUID][]uuid.UUID
	scheduled      map[uuid.UUID]postgres.ScheduledMessageRow
	notifications  map[uuid.UUID]postgres.NotificationRow
	notifyLevels   map[uuid.UUID]map[uuid.UUID]domain.ChannelNotificationLevel
	readState      map[uuid.UUID]map[uuid.UUID]*uuid.UUID
	reactions      map[uuid.UUID]map[string]map[uuid.UUID]time.Time
	memberOrigins  map[uuid.UUID]map[uuid.UUID]httpMemberOrigin
	users          *memStore
}

func newHTTPWorkspaceStore() *httpWorkspaceStore {
	return &httpWorkspaceStore{
		workspaces:     map[uuid.UUID]postgres.WorkspaceRow{},
		members:        map[uuid.UUID]map[uuid.UUID]string{},
		handles:        map[uuid.UUID]map[uuid.UUID]string{},
		aliases:        map[uuid.UUID]map[string]uuid.UUID{},
		icons:          map[uuid.UUID]postgres.WorkspaceIcon{},
		channels:       map[uuid.UUID][]postgres.ChannelRow{},
		channelsByID:   map[uuid.UUID]postgres.ChannelRow{},
		channelMembers: map[uuid.UUID]map[uuid.UUID]string{},
		dmPairs:        map[string]uuid.UUID{},
		slugs:          map[string]uuid.UUID{},
		domains:        map[uuid.UUID]postgres.WorkspaceDomainRow{},
		invites:        map[uuid.UUID]postgres.WorkspaceInviteRow{},
		invitesByToken: map[string]postgres.WorkspaceInviteRow{},
		messages:       map[uuid.UUID]postgres.MessageRow{},
		messagesByCh:   map[uuid.UUID][]uuid.UUID{},
		scheduled:      map[uuid.UUID]postgres.ScheduledMessageRow{},
		notifications:  map[uuid.UUID]postgres.NotificationRow{},
		notifyLevels:   map[uuid.UUID]map[uuid.UUID]domain.ChannelNotificationLevel{},
		readState:      map[uuid.UUID]map[uuid.UUID]*uuid.UUID{},
		reactions:      map[uuid.UUID]map[string]map[uuid.UUID]time.Time{},
	}
}

func (m *httpWorkspaceStore) allocateHandleLocked(workspaceID, userID uuid.UUID, displayName string) string {
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

func (m *httpWorkspaceStore) CreateWorkspace(
	_ context.Context, name, slug string, createdBy uuid.UUID,
) (postgres.CreateWorkspaceResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.slugs[slug]; ok {
		return postgres.CreateWorkspaceResult{}, postgres.ErrSlugConflict
	}
	wsID := id.New()
	ws := postgres.WorkspaceRow{
		ID: wsID, Name: name, Slug: slug, CreatedBy: createdBy, Role: "owner",
	}
	ch := postgres.ChannelRow{
		ID: id.New(), WorkspaceID: wsID, Name: "general", Kind: "channel", CreatedBy: createdBy,
	}
	m.workspaces[wsID] = ws
	m.slugs[slug] = wsID
	m.members[wsID] = map[uuid.UUID]string{createdBy: "owner"}
	displayName := "owner"
	if m.users != nil {
		if u, ok := m.users.byID[createdBy]; ok && u.DisplayName != "" {
			displayName = u.DisplayName
		}
	}
	m.allocateHandleLocked(wsID, createdBy, displayName)
	m.channels[wsID] = []postgres.ChannelRow{ch}
	m.channelsByID[ch.ID] = ch
	return postgres.CreateWorkspaceResult{Workspace: ws, Channels: []postgres.ChannelRow{ch}}, nil
}

func (m *httpWorkspaceStore) ListWorkspacesForUser(_ context.Context, userID uuid.UUID) ([]postgres.WorkspaceRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.WorkspaceRow
	for id, members := range m.members {
		role, ok := members[userID]
		if !ok {
			continue
		}
		ws := m.workspaces[id]
		ws.Role = role
		out = append(out, ws)
	}
	return out, nil
}

func (m *httpWorkspaceStore) GetWorkspaceForUser(
	_ context.Context, workspaceID, userID uuid.UUID,
) (postgres.WorkspaceRow, error) {
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

func (m *httpWorkspaceStore) SlugExists(_ context.Context, slug string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.slugs[slug]
	return ok, nil
}

func (m *httpWorkspaceStore) SlugExistsExcluding(_ context.Context, slug string, excludeID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.slugs[slug]
	if !ok {
		return false, nil
	}
	return id != excludeID, nil
}

func (m *httpWorkspaceStore) UpdateWorkspace(
	_ context.Context, workspaceID uuid.UUID, name, slug string,
) (postgres.WorkspaceRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ws, ok := m.workspaces[workspaceID]
	if !ok {
		return postgres.WorkspaceRow{}, postgres.ErrNotFound
	}
	if other, taken := m.slugs[slug]; taken && other != workspaceID {
		return postgres.WorkspaceRow{}, postgres.ErrSlugConflict
	}
	delete(m.slugs, ws.Slug)
	ws.Name = name
	ws.Slug = slug
	m.workspaces[workspaceID] = ws
	m.slugs[slug] = workspaceID
	return ws, nil
}

func (m *httpWorkspaceStore) SetWorkspaceIcon(
	_ context.Context, workspaceID uuid.UUID, contentType string, data []byte,
) (postgres.WorkspaceRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ws, ok := m.workspaces[workspaceID]
	if !ok {
		return postgres.WorkspaceRow{}, postgres.ErrNotFound
	}
	now := time.Now().UTC()
	m.icons[workspaceID] = postgres.WorkspaceIcon{
		ContentType: contentType, Bytes: append([]byte(nil), data...), UpdatedAt: now,
	}
	ws.HasIcon = true
	ws.IconContentType = contentType
	ws.IconUpdatedAt = &now
	m.workspaces[workspaceID] = ws
	return ws, nil
}

func (m *httpWorkspaceStore) ClearWorkspaceIcon(
	_ context.Context, workspaceID uuid.UUID,
) (postgres.WorkspaceRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ws, ok := m.workspaces[workspaceID]
	if !ok {
		return postgres.WorkspaceRow{}, postgres.ErrNotFound
	}
	delete(m.icons, workspaceID)
	ws.HasIcon = false
	ws.IconContentType = ""
	ws.IconUpdatedAt = nil
	m.workspaces[workspaceID] = ws
	return ws, nil
}

func (m *httpWorkspaceStore) GetWorkspaceIcon(
	_ context.Context, workspaceID uuid.UUID,
) (postgres.WorkspaceIcon, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	icon, ok := m.icons[workspaceID]
	if !ok {
		return postgres.WorkspaceIcon{}, postgres.ErrNotFound
	}
	return icon, nil
}

func (m *httpWorkspaceStore) DeleteWorkspace(_ context.Context, workspaceID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ws, ok := m.workspaces[workspaceID]
	if !ok {
		return postgres.ErrNotFound
	}
	delete(m.slugs, ws.Slug)
	delete(m.workspaces, workspaceID)
	delete(m.members, workspaceID)
	delete(m.icons, workspaceID)
	delete(m.channels, workspaceID)
	return nil
}

func (m *httpWorkspaceStore) ListChannelsForWorkspace(
	_ context.Context, workspaceID, userID uuid.UUID,
) ([]postgres.ChannelRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	members, ok := m.members[workspaceID]
	if !ok {
		return nil, postgres.ErrNotFound
	}
	if _, ok := members[userID]; !ok {
		return nil, postgres.ErrNotFound
	}
	var out []postgres.ChannelRow
	for _, ch := range m.channels[workspaceID] {
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

func (m *httpWorkspaceStore) GetUserByEmail(ctx context.Context, email string) (postgres.UserRow, error) {
	if m.users == nil {
		return postgres.UserRow{}, postgres.ErrNotFound
	}
	return m.users.GetUserByEmail(ctx, email)
}

func (m *httpWorkspaceStore) GetUserByID(ctx context.Context, userID uuid.UUID) (postgres.UserRow, error) {
	if m.users == nil {
		return postgres.UserRow{}, postgres.ErrNotFound
	}
	return m.users.GetUserByID(ctx, userID)
}

func (m *httpWorkspaceStore) ListUserNotificationLevels(
	ctx context.Context, userIDs []uuid.UUID,
) (map[uuid.UUID]domain.NotificationLevel, error) {
	out := map[uuid.UUID]domain.NotificationLevel{}
	for _, id := range userIDs {
		row, err := m.GetUserByID(ctx, id)
		if err != nil {
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

func (m *httpWorkspaceStore) IsWorkspaceMember(_ context.Context, workspaceID, userID uuid.UUID) (bool, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	role, ok := m.members[workspaceID][userID]
	if !ok {
		return false, "", nil
	}
	return true, role, nil
}

func (m *httpWorkspaceStore) CreateChannel(
	_ context.Context, workspaceID, createdBy uuid.UUID, name, topic string, isPrivate bool,
) (postgres.ChannelRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ch := range m.channels[workspaceID] {
		if ch.Name == name {
			return postgres.ChannelRow{}, postgres.ErrChannelNameConflict
		}
	}
	now := time.Now().UTC()
	row := postgres.ChannelRow{
		ID: id.New(), WorkspaceID: workspaceID, Name: name, Topic: topic,
		IsPrivate: isPrivate, Kind: "channel", CreatedBy: createdBy, CreatedAt: now, UpdatedAt: now,
	}
	m.channels[workspaceID] = append(m.channels[workspaceID], row)
	m.channelsByID[row.ID] = row
	if isPrivate {
		m.channelMembers[row.ID] = map[uuid.UUID]string{createdBy: string(domain.ChannelMemberRoleAdmin)}
	}
	return row, nil
}

func (m *httpWorkspaceStore) GetChannel(_ context.Context, channelID uuid.UUID) (postgres.ChannelRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.channelsByID[channelID]
	if !ok {
		return postgres.ChannelRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func dmPairKey(workspaceID, a, b uuid.UUID) string {
	left, right := a, b
	if a.String() > b.String() {
		left, right = b, a
	}
	return workspaceID.String() + "|" + left.String() + "|" + right.String()
}

func (m *httpWorkspaceStore) enrichDMPeer(row postgres.ChannelRow, viewerID uuid.UUID) postgres.ChannelRow {
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
	if m.users == nil {
		row.PeerUserID = &peerID
		return row
	}
	peer, err := m.users.GetUserByID(context.Background(), peerID)
	if err != nil {
		row.PeerUserID = &peerID
		return row
	}
	row.PeerUserID = &peerID
	row.PeerDisplayName = peer.DisplayName
	row.PeerHasAvatar = peer.HasAvatar
	row.PeerAvatarUpdatedAt = peer.AvatarUpdatedAt
	if handles := m.handles[row.WorkspaceID]; handles != nil {
		row.PeerHandle = handles[peerID]
	}
	return row
}

func (m *httpWorkspaceStore) GetDMChannelForUser(
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

func (m *httpWorkspaceStore) FindDMChannel(
	_ context.Context, workspaceID, userA, userB uuid.UUID,
) (postgres.ChannelRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	chID, ok := m.dmPairs[dmPairKey(workspaceID, userA, userB)]
	if !ok {
		return postgres.ChannelRow{}, postgres.ErrNotFound
	}
	row, ok := m.channelsByID[chID]
	if !ok {
		return postgres.ChannelRow{}, postgres.ErrNotFound
	}
	return m.enrichDMPeer(row, userA), nil
}

func (m *httpWorkspaceStore) CreateDMChannel(
	_ context.Context, workspaceID, createdBy, peerID uuid.UUID,
) (postgres.ChannelRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := dmPairKey(workspaceID, createdBy, peerID)
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
		Name:      "dm_" + left.String() + "_" + right.String(),
		IsPrivate: true, Kind: "dm", CreatedBy: createdBy, CreatedAt: now, UpdatedAt: now,
	}
	m.channels[workspaceID] = append(m.channels[workspaceID], row)
	m.channelsByID[row.ID] = row
	m.channelMembers[row.ID] = map[uuid.UUID]string{
		createdBy: string(domain.ChannelMemberRoleMember),
		peerID:    string(domain.ChannelMemberRoleMember),
	}
	m.dmPairs[key] = row.ID
	return m.enrichDMPeer(row, createdBy), nil
}

func (m *httpWorkspaceStore) UpdateChannel(
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
	list := m.channels[row.WorkspaceID]
	for i, ch := range list {
		if ch.ID == channelID {
			list[i] = row
		}
	}
	m.channels[row.WorkspaceID] = list
	if isPrivate {
		if m.channelMembers[channelID] == nil {
			m.channelMembers[channelID] = map[uuid.UUID]string{}
		}
		m.channelMembers[channelID][updatedBy] = string(domain.ChannelMemberRoleAdmin)
		m.channelMembers[channelID][row.CreatedBy] = string(domain.ChannelMemberRoleAdmin)
	}
	return row, nil
}

func (m *httpWorkspaceStore) DeleteChannel(_ context.Context, channelID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.channelsByID[channelID]
	if !ok {
		return postgres.ErrNotFound
	}
	delete(m.channelsByID, channelID)
	delete(m.channelMembers, channelID)
	filtered := make([]postgres.ChannelRow, 0)
	for _, ch := range m.channels[row.WorkspaceID] {
		if ch.ID != channelID {
			filtered = append(filtered, ch)
		}
	}
	m.channels[row.WorkspaceID] = filtered
	return nil
}

func (m *httpWorkspaceStore) IsChannelMember(_ context.Context, channelID, userID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.channelMembers[channelID][userID]
	return ok, nil
}

func (m *httpWorkspaceStore) ListChannelMemberIDs(_ context.Context, channelID uuid.UUID) ([]uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []uuid.UUID
	for userID := range m.channelMembers[channelID] {
		out = append(out, userID)
	}
	return out, nil
}

func (m *httpWorkspaceStore) GetChannelNotificationLevel(
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

func (m *httpWorkspaceStore) UpsertChannelNotificationLevel(
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

func (m *httpWorkspaceStore) ListChannelNotificationLevels(
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

func (m *httpWorkspaceStore) AddChannelMember(_ context.Context, channelID, userID uuid.UUID, role domain.ChannelMemberRole) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.channelMembers[channelID] == nil {
		m.channelMembers[channelID] = map[uuid.UUID]string{}
	}
	m.channelMembers[channelID][userID] = string(role)
	return nil
}

func (m *httpWorkspaceStore) RemoveChannelMember(_ context.Context, channelID, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.channelMembers[channelID][userID]; !ok {
		return postgres.ErrNotFound
	}
	delete(m.channelMembers[channelID], userID)
	return nil
}

func (m *httpWorkspaceStore) CreateWorkspaceInvite(
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

func (m *httpWorkspaceStore) ListPendingWorkspaceInvites(_ context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceInviteRow, error) {
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

func (m *httpWorkspaceStore) GetWorkspaceInviteByToken(_ context.Context, token string) (postgres.WorkspaceInviteRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.invitesByToken[token]
	if !ok {
		return postgres.WorkspaceInviteRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *httpWorkspaceStore) AcceptWorkspaceInvite(_ context.Context, inviteID uuid.UUID) error {
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

func (m *httpWorkspaceStore) InsertMessage(
	_ context.Context, channelID, authorID uuid.UUID, body, contentType string,
) (postgres.MessageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	name := ""
	hasAvatar := false
	var avatarUpdated *time.Time
	if m.users != nil {
		if u, ok := m.users.byID[authorID]; ok {
			name = u.DisplayName
			hasAvatar = u.HasAvatar
			avatarUpdated = u.AvatarUpdatedAt
		}
	}
	handle := ""
	if ch, ok := m.channelsByID[channelID]; ok {
		handle = m.handles[ch.WorkspaceID][authorID]
	}
	row := postgres.MessageRow{
		ID: id.New(), ChannelID: channelID, AuthorID: authorID,
		AuthorName: name, AuthorHandle: handle,
		AuthorHasAvatar: hasAvatar, AuthorAvatarUpdated: avatarUpdated,
		Body: body, ContentType: contentType, CreatedAt: now, UpdatedAt: now,
	}
	m.messages[row.ID] = row
	m.messagesByCh[channelID] = append(m.messagesByCh[channelID], row.ID)
	return row, nil
}

func (m *httpWorkspaceStore) ListMessages(
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

func (m *httpWorkspaceStore) GetMessage(_ context.Context, messageID uuid.UUID) (postgres.MessageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.messages[messageID]
	if !ok {
		return postgres.MessageRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *httpWorkspaceStore) UpdateMessageBody(
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

func (m *httpWorkspaceStore) DeleteMessage(_ context.Context, messageID uuid.UUID) error {
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

func (m *httpWorkspaceStore) UpsertChannelReadState(
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

func (m *httpWorkspaceStore) GetChannelReadState(
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

func (m *httpWorkspaceStore) GetPreviousMessageID(
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

func (m *httpWorkspaceStore) GetLatestMessageID(_ context.Context, channelID uuid.UUID) (*uuid.UUID, error) {
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

func (m *httpWorkspaceStore) GetChannelUnreadSummary(
	_ context.Context, userID, channelID uuid.UUID,
) (int, *uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count, first := m.unreadSummaryLocked(userID, channelID)
	return count, first, nil
}

func (m *httpWorkspaceStore) AddMessageReaction(
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

func (m *httpWorkspaceStore) RemoveMessageReaction(
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

func (m *httpWorkspaceStore) HasMessageReaction(
	_ context.Context, messageID, userID uuid.UUID, emoji string,
) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.reactions[messageID][emoji][userID]
	return ok, nil
}

func (m *httpWorkspaceStore) ListReactionsForMessages(
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

func (m *httpWorkspaceStore) ensureChannelReadBaselineLocked(userID, channelID uuid.UUID) {
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

func (m *httpWorkspaceStore) unreadSummaryLocked(userID, channelID uuid.UUID) (int, *uuid.UUID) {
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

func (m *httpWorkspaceStore) InsertScheduledMessage(
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
	return row, nil
}

func (m *httpWorkspaceStore) ListScheduledMessages(_ context.Context, channelID uuid.UUID) ([]postgres.ScheduledMessageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.ScheduledMessageRow
	for _, row := range m.scheduled {
		if row.ChannelID == channelID && row.Status == string(domain.ScheduledPending) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (m *httpWorkspaceStore) GetScheduledMessage(_ context.Context, scheduledID uuid.UUID) (postgres.ScheduledMessageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.scheduled[scheduledID]
	if !ok {
		return postgres.ScheduledMessageRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *httpWorkspaceStore) CancelScheduledMessage(_ context.Context, scheduledID uuid.UUID) error {
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

func (m *httpWorkspaceStore) ClaimAndPublishDueScheduledMessages(
	_ context.Context, now time.Time, limit int,
) ([]postgres.PublishedScheduledMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var published []postgres.PublishedScheduledMessage
	for scheduledID, row := range m.scheduled {
		if limit > 0 && len(published) >= limit {
			break
		}
		if row.Status != string(domain.ScheduledPending) || row.SendAt.After(now) {
			continue
		}
		msgID := id.New()
		msg := postgres.MessageRow{
			ID: msgID, ChannelID: row.ChannelID, AuthorID: row.AuthorID,
			Body: row.Body, ContentType: row.ContentType, CreatedAt: now, UpdatedAt: now,
		}
		m.messages[msgID] = msg
		m.messagesByCh[row.ChannelID] = append(m.messagesByCh[row.ChannelID], msgID)
		row.Status = string(domain.ScheduledSent)
		row.SentMessageID = &msgID
		m.scheduled[scheduledID] = row
		published = append(published, postgres.PublishedScheduledMessage{
			ID: msgID, ChannelID: row.ChannelID, AuthorID: row.AuthorID, Body: row.Body,
		})
	}
	return published, nil
}

func (m *httpWorkspaceStore) formerHandlesLocked(workspaceID, userID uuid.UUID) []string {
	var out []string
	for handle, owner := range m.aliases[workspaceID] {
		if owner == userID {
			out = append(out, handle)
		}
	}
	return out
}

func (m *httpWorkspaceStore) ListWorkspaceMembers(_ context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceMemberInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.WorkspaceMemberInfo
	for userID, role := range m.members[workspaceID] {
		name := ""
		if m.users != nil {
			if u, ok := m.users.byID[userID]; ok {
				name = u.DisplayName
			}
		}
		hasAvatar := false
		var avatarUpdated *time.Time
		if u, ok := m.users.byID[userID]; ok {
			hasAvatar = u.HasAvatar
			avatarUpdated = u.AvatarUpdatedAt
		}
		out = append(out, postgres.WorkspaceMemberInfo{
			UserID: userID, DisplayName: name, Handle: m.handles[workspaceID][userID],
			FormerHandles: m.formerHandlesLocked(workspaceID, userID), Role: role,
			HasAvatar: hasAvatar, AvatarUpdatedAt: avatarUpdated,
		})
	}
	return out, nil
}

func (m *httpWorkspaceStore) GetWorkspaceMember(
	_ context.Context, workspaceID, userID uuid.UUID,
) (postgres.WorkspaceMemberInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	role, ok := m.members[workspaceID][userID]
	if !ok {
		return postgres.WorkspaceMemberInfo{}, postgres.ErrNotFound
	}
	name := ""
	hasAvatar := false
	var avatarUpdated *time.Time
	if m.users != nil {
		if u, ok := m.users.byID[userID]; ok {
			name = u.DisplayName
			hasAvatar = u.HasAvatar
			avatarUpdated = u.AvatarUpdatedAt
		}
	}
	return postgres.WorkspaceMemberInfo{
		UserID: userID, DisplayName: name, Handle: m.handles[workspaceID][userID],
		FormerHandles: m.formerHandlesLocked(workspaceID, userID), Role: role,
		HasAvatar: hasAvatar, AvatarUpdatedAt: avatarUpdated,
	}, nil
}

func (m *httpWorkspaceStore) MemberHandleExists(
	_ context.Context, workspaceID uuid.UUID, handle string, excludeUserID uuid.UUID,
) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for userID, h := range m.handles[workspaceID] {
		if userID != excludeUserID && h == handle {
			return true, nil
		}
	}
	return false, nil
}

func (m *httpWorkspaceStore) UpdateWorkspaceMemberHandle(
	_ context.Context, workspaceID, userID uuid.UUID, handle string,
) (postgres.WorkspaceMemberInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	role, ok := m.members[workspaceID][userID]
	if !ok {
		return postgres.WorkspaceMemberInfo{}, postgres.ErrNotFound
	}
	for otherID, h := range m.handles[workspaceID] {
		if otherID != userID && h == handle {
			return postgres.WorkspaceMemberInfo{}, postgres.ErrHandleConflict
		}
	}
	if m.handles[workspaceID] == nil {
		m.handles[workspaceID] = map[uuid.UUID]string{}
	}
	if m.aliases[workspaceID] == nil {
		m.aliases[workspaceID] = map[string]uuid.UUID{}
	}
	oldHandle := m.handles[workspaceID][userID]
	if oldHandle != handle {
		delete(m.aliases[workspaceID], handle)
		m.aliases[workspaceID][oldHandle] = userID
		m.handles[workspaceID][userID] = handle
	}
	name := ""
	hasAvatar := false
	var avatarUpdated *time.Time
	if m.users != nil {
		if u, ok := m.users.byID[userID]; ok {
			name = u.DisplayName
			hasAvatar = u.HasAvatar
			avatarUpdated = u.AvatarUpdatedAt
		}
	}
	return postgres.WorkspaceMemberInfo{
		UserID: userID, DisplayName: name, Handle: handle,
		FormerHandles: m.formerHandlesLocked(workspaceID, userID), Role: role,
		HasAvatar: hasAvatar, AvatarUpdatedAt: avatarUpdated,
	}, nil
}

func (m *httpWorkspaceStore) CreateNotification(
	_ context.Context, in postgres.CreateNotificationInput,
) (postgres.NotificationRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	row := postgres.NotificationRow{
		ID: id.New(), UserID: in.UserID, ActorID: in.ActorID, Kind: string(in.Kind),
		WorkspaceID: in.WorkspaceID, ChannelID: in.ChannelID, MessageID: in.MessageID,
		Body: in.Body, CreatedAt: now,
	}
	m.notifications[row.ID] = row
	return row, nil
}

func (m *httpWorkspaceStore) enrichNotificationChannelLocked(
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

func (m *httpWorkspaceStore) ListNotifications(
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

func (m *httpWorkspaceStore) CountUnreadNotifications(_ context.Context, userID uuid.UUID) (int, error) {
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

func (m *httpWorkspaceStore) MarkNotificationRead(_ context.Context, userID, notificationID uuid.UUID) error {
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

func (m *httpWorkspaceStore) MarkAllNotificationsRead(_ context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	for nid, row := range m.notifications {
		if row.UserID == userID && row.ReadAt == nil {
			row.ReadAt = &now
			m.notifications[nid] = row
		}
	}
	return nil
}

func (m *httpWorkspaceStore) CreateWorkspaceDomain(
	_ context.Context, workspaceID, createdBy uuid.UUID, domainName, verificationToken string,
) (postgres.WorkspaceDomainRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, d := range m.domains {
		if d.WorkspaceID == workspaceID && d.Domain == domainName {
			return postgres.WorkspaceDomainRow{}, postgres.ErrDomainConflict
		}
	}
	now := time.Now().UTC()
	row := postgres.WorkspaceDomainRow{
		ID: id.New(), WorkspaceID: workspaceID, Domain: domainName,
		VerificationToken: verificationToken, CreatedBy: createdBy,
		CreatedAt: now, UpdatedAt: now,
	}
	m.domains[row.ID] = row
	return row, nil
}

func (m *httpWorkspaceStore) ListWorkspaceDomains(_ context.Context, workspaceID uuid.UUID) ([]postgres.WorkspaceDomainRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []postgres.WorkspaceDomainRow
	for _, d := range m.domains {
		if d.WorkspaceID == workspaceID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (m *httpWorkspaceStore) GetWorkspaceDomain(
	_ context.Context, workspaceID, domainID uuid.UUID,
) (postgres.WorkspaceDomainRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.domains[domainID]
	if !ok || row.WorkspaceID != workspaceID {
		return postgres.WorkspaceDomainRow{}, postgres.ErrNotFound
	}
	return row, nil
}

func (m *httpWorkspaceStore) MarkWorkspaceDomainVerified(
	_ context.Context, workspaceID, domainID uuid.UUID,
) (postgres.WorkspaceDomainRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.domains[domainID]
	if !ok || row.WorkspaceID != workspaceID {
		return postgres.WorkspaceDomainRow{}, postgres.ErrNotFound
	}
	for _, d := range m.domains {
		if d.Domain == row.Domain && d.VerifiedAt != nil && d.WorkspaceID != workspaceID {
			return postgres.WorkspaceDomainRow{}, postgres.ErrDomainVerifiedConflict
		}
	}
	now := time.Now().UTC()
	row.VerifiedAt = &now
	row.UpdatedAt = now
	m.domains[domainID] = row
	return row, nil
}

func (m *httpWorkspaceStore) UpdateWorkspaceDomainAutoJoin(
	_ context.Context, workspaceID, domainID uuid.UUID, autoJoin bool,
) (postgres.WorkspaceDomainRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.domains[domainID]
	if !ok || row.WorkspaceID != workspaceID || row.VerifiedAt == nil {
		return postgres.WorkspaceDomainRow{}, postgres.ErrNotFound
	}
	row.AutoJoin = autoJoin
	row.UpdatedAt = time.Now().UTC()
	m.domains[domainID] = row
	return row, nil
}

func (m *httpWorkspaceStore) DeleteWorkspaceDomain(_ context.Context, workspaceID, domainID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.domains[domainID]
	if !ok || row.WorkspaceID != workspaceID {
		return postgres.ErrNotFound
	}
	delete(m.domains, domainID)
	return nil
}

func (m *httpWorkspaceStore) DomainVerifiedElsewhere(
	_ context.Context, domainName string, excludeWorkspaceID uuid.UUID,
) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, d := range m.domains {
		if d.Domain == domainName && d.VerifiedAt != nil && d.WorkspaceID != excludeWorkspaceID {
			return true, nil
		}
	}
	return false, nil
}

func (m *httpWorkspaceStore) ListAutoJoinWorkspaceIDsByDomain(_ context.Context, domainName string) ([]uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []uuid.UUID
	for _, d := range m.domains {
		if d.Domain == domainName && d.VerifiedAt != nil && d.AutoJoin {
			out = append(out, d.WorkspaceID)
		}
	}
	return out, nil
}

func (m *httpWorkspaceStore) AddWorkspaceMember(
	_ context.Context, workspaceID, userID uuid.UUID, role domain.WorkspaceRole,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members[workspaceID] == nil {
		m.members[workspaceID] = map[uuid.UUID]string{}
	}
	if _, ok := m.members[workspaceID][userID]; !ok {
		m.members[workspaceID][userID] = string(role)
		displayName := "member"
		if m.users != nil {
			if u, ok := m.users.byID[userID]; ok && u.DisplayName != "" {
				displayName = u.DisplayName
			}
		}
		m.allocateHandleLocked(workspaceID, userID, displayName)
	}
	return nil
}

func TestWorkspaceCreateAndList(t *testing.T) {
	t.Parallel()
	h := testServer()

	signupBody := []byte(`{"email":"ws@example.com","password":"ValidPass1234","displayName":"WS"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(signupBody))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("signup = %d", rec.Code)
	}
	var auth authResponse
	if err := json.NewDecoder(rec.Body).Decode(&auth); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list empty = %d", rec.Code)
	}
	var list listWorkspacesResponse
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list.Workspaces) != 0 {
		t.Fatalf("expected empty list, got %+v", list.Workspaces)
	}

	createBody := []byte(`{"name":"Kilobyte"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d body=%s", rec.Code, rec.Body.String())
	}
	var created createWorkspaceResponse
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Workspace.Slug != "kilobyte" || len(created.Channels) != 1 || created.Channels[0].Name != "general" {
		t.Fatalf("created = %+v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+created.Workspace.ID+"/channels", nil)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("channels = %d", rec.Code)
	}

	otherBody := []byte(`{"email":"other@example.com","password":"ValidPass1234","displayName":"Other"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(otherBody))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("other signup = %d", rec.Code)
	}
	var other authResponse
	if err := json.NewDecoder(rec.Body).Decode(&other); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+created.Workspace.ID+"/channels", nil)
	req.Header.Set("Authorization", "Bearer "+other.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("non-member channels = %d, want 404", rec.Code)
	}

	renameBody := []byte(`{"name":"northwind"}`)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/"+created.Workspace.ID, bytes.NewReader(renameBody))
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("rename = %d body=%s", rec.Code, rec.Body.String())
	}
	var renamed getWorkspaceResponse
	if err := json.NewDecoder(rec.Body).Decode(&renamed); err != nil {
		t.Fatal(err)
	}
	if renamed.Workspace.Name != "Northwind" {
		t.Fatalf("renamed name = %q", renamed.Workspace.Name)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/"+created.Workspace.ID, nil)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d", rec.Code)
	}
}

func TestWorkspaceHandleAPI(t *testing.T) {
	t.Parallel()
	h := testServer()

	signupBody := []byte(`{"email":"handle@example.com","password":"ValidPass1234","displayName":"Ada Lovelace"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(signupBody))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("signup = %d", rec.Code)
	}
	var auth authResponse
	if err := json.NewDecoder(rec.Body).Decode(&auth); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader([]byte(`{"name":"Lab"}`)))
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d", rec.Code)
	}
	var created createWorkspaceResponse
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+created.Workspace.ID+"/me", nil)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("me = %d body=%s", rec.Code, rec.Body.String())
	}
	var me workspaceMemberResponse
	if err := json.NewDecoder(rec.Body).Decode(&me); err != nil {
		t.Fatal(err)
	}
	if me.Member.Handle != "ada_lovelace" {
		t.Fatalf("handle = %q", me.Member.Handle)
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/"+created.Workspace.ID+"/me", bytes.NewReader([]byte(`{"handle":"ada"}`)))
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch me = %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+created.Workspace.ID+"/members", nil)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("members = %d", rec.Code)
	}
	var members listWorkspaceMembersResponse
	if err := json.NewDecoder(rec.Body).Decode(&members); err != nil {
		t.Fatal(err)
	}
	if len(members.Members) != 1 || members.Members[0].Handle != "ada" {
		t.Fatalf("members = %+v", members.Members)
	}
}

func TestOpenDMAPI(t *testing.T) {
	t.Parallel()
	h := testServer()

	signupA := []byte(`{"email":"dma@example.com","password":"ValidPass1234","displayName":"DM A"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(signupA))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("signup A = %d", rec.Code)
	}
	var authA authResponse
	if err := json.NewDecoder(rec.Body).Decode(&authA); err != nil {
		t.Fatal(err)
	}

	signupB := []byte(`{"email":"dmb@example.com","password":"ValidPass1234","displayName":"DM B"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(signupB))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("signup B = %d", rec.Code)
	}
	var authB authResponse
	if err := json.NewDecoder(rec.Body).Decode(&authB); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader([]byte(`{"name":"DM Lab"}`)))
	req.Header.Set("Authorization", "Bearer "+authA.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d body=%s", rec.Code, rec.Body.String())
	}
	var created createWorkspaceResponse
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	inviteBody := []byte(`{"email":"dmb@example.com","role":"member"}`)
	req = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/workspaces/"+created.Workspace.ID+"/invites",
		bytes.NewReader(inviteBody),
	)
	req.Header.Set("Authorization", "Bearer "+authA.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("invite = %d body=%s", rec.Code, rec.Body.String())
	}

	selfBody := []byte(fmt.Sprintf(`{"userId":%q}`, authA.User.ID))
	req = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/workspaces/"+created.Workspace.ID+"/dms",
		bytes.NewReader(selfBody),
	)
	req.Header.Set("Authorization", "Bearer "+authA.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("self DM = %d, want 400", rec.Code)
	}

	openBody := []byte(fmt.Sprintf(`{"userId":%q}`, authB.User.ID))
	req = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/workspaces/"+created.Workspace.ID+"/dms",
		bytes.NewReader(openBody),
	)
	req.Header.Set("Authorization", "Bearer "+authA.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("open DM = %d body=%s", rec.Code, rec.Body.String())
	}
	var opened struct {
		Channel channelJSON `json:"channel"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&opened); err != nil {
		t.Fatal(err)
	}
	if !opened.Channel.IsDM || opened.Channel.PeerUserID != authB.User.ID {
		t.Fatalf("opened = %+v", opened.Channel)
	}

	req = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/workspaces/"+created.Workspace.ID+"/dms",
		bytes.NewReader(openBody),
	)
	req.Header.Set("Authorization", "Bearer "+authA.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("reopen DM = %d", rec.Code)
	}
	var reopened struct {
		Channel channelJSON `json:"channel"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&reopened); err != nil {
		t.Fatal(err)
	}
	if reopened.Channel.ID != opened.Channel.ID {
		t.Fatalf("expected same channel id, got %s vs %s", opened.Channel.ID, reopened.Channel.ID)
	}

	msgBody := []byte(`{"body":"hello dm"}`)
	req = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/channels/"+opened.Channel.ID+"/messages",
		bytes.NewReader(msgBody),
	)
	req.Header.Set("Authorization", "Bearer "+authA.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("post message = %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+opened.Channel.ID+"/messages", nil)
	req.Header.Set("Authorization", "Bearer "+authB.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list messages = %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+created.Workspace.ID+"/channels", nil)
	req.Header.Set("Authorization", "Bearer "+authA.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list channels = %d", rec.Code)
	}
	var channels listChannelsResponse
	if err := json.NewDecoder(rec.Body).Decode(&channels); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, ch := range channels.Channels {
		if ch.ID == opened.Channel.ID {
			found = true
			if !ch.IsDM || ch.PeerUserID != authB.User.ID {
				t.Fatalf("listed DM = %+v", ch)
			}
		}
	}
	if !found {
		t.Fatalf("DM missing from list: %+v", channels.Channels)
	}

	req = httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/channels/"+opened.Channel.ID,
		bytes.NewReader([]byte(`{"name":"nope","topic":"","isPrivate":true}`)),
	)
	req.Header.Set("Authorization", "Bearer "+authA.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("rename DM = %d, want 400", rec.Code)
	}
}

func (m *httpWorkspaceStore) CountWorkspaceOwners(_ context.Context, workspaceID uuid.UUID) (int, error) {
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

func (m *httpWorkspaceStore) UpdateWorkspaceMemberRole(
	_ context.Context, workspaceID, userID uuid.UUID, role domain.WorkspaceRole,
) (postgres.WorkspaceMemberInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.members[workspaceID][userID]; !ok {
		return postgres.WorkspaceMemberInfo{}, postgres.ErrNotFound
	}
	m.members[workspaceID][userID] = string(role)
	name := ""
	if m.users != nil {
		if u, err := m.users.GetUserByID(context.Background(), userID); err == nil {
			name = u.DisplayName
		}
	}
	return postgres.WorkspaceMemberInfo{
		UserID: userID, DisplayName: name, Handle: m.handles[workspaceID][userID],
		Role: string(role), Kind: "member",
	}, nil
}

func (m *httpWorkspaceStore) RemoveWorkspaceMember(_ context.Context, workspaceID, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.members[workspaceID][userID]; !ok {
		return postgres.ErrNotFound
	}
	delete(m.members[workspaceID], userID)
	delete(m.handles[workspaceID], userID)
	return nil
}

func (m *httpWorkspaceStore) TransferWorkspaceOwnership(
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

func (m *httpWorkspaceStore) DeleteWorkspaceInvite(_ context.Context, workspaceID, inviteID uuid.UUID) error {
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

func (m *httpWorkspaceStore) FindHomeWorkspaceForUser(_ context.Context, userID uuid.UUID) (uuid.UUID, string, error) {
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

func (m *httpWorkspaceStore) SetWorkspaceMemberOrigin(
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
		m.memberOrigins = map[uuid.UUID]map[uuid.UUID]httpMemberOrigin{}
	}
	if m.memberOrigins[workspaceID] == nil {
		m.memberOrigins[workspaceID] = map[uuid.UUID]httpMemberOrigin{}
	}
	m.memberOrigins[workspaceID][userID] = httpMemberOrigin{
		Kind: kind, HomeWorkspaceID: homeWorkspaceID, HomeWorkspaceName: homeWorkspaceName,
		HomeServerID: homeServerID, HomeWorkspaceRemoteID: homeWorkspaceRemoteID,
		HomeWorkspaceIconURL: homeWorkspaceIconURL,
	}
	return nil
}

type httpMemberOrigin struct {
	Kind                  string
	HomeWorkspaceID       *uuid.UUID
	HomeWorkspaceName     string
	HomeServerID          string
	HomeWorkspaceRemoteID string
	HomeWorkspaceIconURL  string
}
