package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kilobyteno/dagr-chat/internal/service"
)

func testLoggingServer() *Server {
	return &Server{logger: slog.New(slog.DiscardHandler)}
}

func decodeAPIError(t *testing.T, rec *httptest.ResponseRecorder) apiError {
	t.Helper()
	var body apiErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Error
}

func TestWriteAuthErrorMapped(t *testing.T) {
	t.Parallel()
	s := testLoggingServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	s.writeAuthError(rec, req, service.ErrInvalidCredentials)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
	got := decodeAPIError(t, rec)
	if got.Code != "invalid_credentials" {
		t.Fatalf("code = %s", got.Code)
	}
	if got.Message != "Invalid email or password" {
		t.Fatalf("message = %s", got.Message)
	}
}

func TestWriteAuthErrorUnmapped(t *testing.T) {
	t.Parallel()
	s := testLoggingServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	s.writeAuthError(rec, req, fmt.Errorf("insert user: connection refused"))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	got := decodeAPIError(t, rec)
	if got.Code != "internal_error" {
		t.Fatalf("code = %s", got.Code)
	}
	if got.Message != "Something went wrong" {
		t.Fatalf("message = %s", got.Message)
	}
	if strings.Contains(rec.Body.String(), "connection refused") {
		t.Fatal("leaked Go error string")
	}
}

func TestRequestIDHeader(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	testServer().ServeHTTP(rec, req)
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("missing X-Request-Id")
	}
}
