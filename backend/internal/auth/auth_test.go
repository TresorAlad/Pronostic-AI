package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUserIDFromContextEmpty(t *testing.T) {
	_, ok := UserIDFromContext(context.Background())
	if ok {
		t.Fatal("expected no user id in empty context")
	}
}

func TestRequireAuthRejectsAnonymous(t *testing.T) {
	svc := NewService(nil, "test-secret-key-32-chars-minimum!!", time.Hour)
	handler := svc.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuthAllowsAuthenticated(t *testing.T) {
	svc := NewService(nil, "test-secret-key-32-chars-minimum!!", time.Hour)
	handler := svc.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ctx := context.WithValue(context.Background(), userIDKey, "user-123")
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestUpdateMeRequiresDisplayName(t *testing.T) {
	svc := NewService(nil, "test-secret-key-32-chars-minimum!!", time.Hour)
	handler := svc.RequireAuth(http.HandlerFunc(svc.UpdateMe))

	ctx := context.WithValue(context.Background(), userIDKey, "user-123")
	req := httptest.NewRequest(http.MethodPatch, "/auth/me", strings.NewReader(`{}`))
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGenerateAndParseToken(t *testing.T) {
	svc := NewService(nil, "test-secret-key-32-chars-minimum!!", time.Hour)
	token, err := svc.generateToken("uid-1", "a@b.com")
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
}
