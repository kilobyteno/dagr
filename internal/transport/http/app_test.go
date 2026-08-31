package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kilobyteno/dagr-chat/internal/domain"
)

func TestIncomingWebhookHTTPFlow(t *testing.T) {
	t.Parallel()
	h := testServer()

	signupBody := []byte(`{"email":"hooks@example.com","password":"ValidPass1234","displayName":"Hooks"}`)
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

	req = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader([]byte(`{"name":"Hooks"}`)))
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d body=%s", rec.Code, rec.Body.String())
	}
	var created createWorkspaceResponse
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if len(created.Channels) == 0 {
		t.Fatal("expected general channel")
	}
	channelID := created.Channels[0].ID

	req = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+created.Workspace.ID+"/apps", nil)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list apps = %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/channels/"+channelID+"/apps/incoming-webhooks",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("enable webhook = %d body=%s", rec.Code, rec.Body.String())
	}
	var enabled struct {
		Webhook incomingWebhookJSON `json:"webhook"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled.Webhook.URL == "" {
		t.Fatal("expected webhook URL once")
	}

	req = httptest.NewRequest(http.MethodPost, enabled.Webhook.URL, bytes.NewReader([]byte(
		`{"text":"Build passed","embeds":[{"title":"CI","color":"#22c55e"}]}`,
	)))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("post webhook = %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+channelID+"/messages", nil)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list messages = %d", rec.Code)
	}
	var listed struct {
		Messages []messageJSON `json:"messages"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Messages) != 1 || listed.Messages[0].ContentType != domain.ContentTypeRich {
		t.Fatalf("messages = %+v", listed.Messages)
	}
	if listed.Messages[0].Payload == nil || listed.Messages[0].Payload.Text != "Build passed" {
		t.Fatalf("payload = %+v", listed.Messages[0].Payload)
	}

	req = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/channels/"+channelID+"/apps/incoming-webhooks/rotate",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("rotate = %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, enabled.Webhook.URL, bytes.NewReader([]byte(`{"text":"old"}`)))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("old token = %d", rec.Code)
	}
}
