package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shire-studio/webhookhub-cli/internal/auth"
)

func TestAuthLogin_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cli/me" {
			t.Errorf("path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"email":"dev@example.com","plan":"indie"}}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)

	out := &bytes.Buffer{}
	err := runAuthLogin(authLoginOpts{
		BaseURL:   server.URL,
		Store:     store,
		ReadToken: func() (string, error) { return "whk_test", nil },
		Out:       out,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("expected token saved: %v", err)
	}
	if cfg.Token != "whk_test" {
		t.Errorf("got token %q", cfg.Token)
	}

	if !strings.Contains(out.String(), "dev@example.com") {
		t.Errorf("expected confirmation, got %q", out.String())
	}
}

func TestAuthLogin_RejectsBadToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
	}))
	defer server.Close()

	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)

	out := &bytes.Buffer{}
	err := runAuthLogin(authLoginOpts{
		BaseURL:   server.URL,
		Store:     store,
		ReadToken: func() (string, error) { return "whk_bad", nil },
		Out:       out,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "token") {
		t.Errorf("expected error to mention token, got %v", err)
	}

	if _, err := store.Load(); err != auth.ErrNotLoggedIn {
		t.Errorf("expected NO config saved, got %v", err)
	}
}

func TestAuthLogin_RejectsEmptyToken(t *testing.T) {
	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)

	out := &bytes.Buffer{}
	err := runAuthLogin(authLoginOpts{
		BaseURL:   "http://example.invalid",
		Store:     store,
		ReadToken: func() (string, error) { return "", nil },
		Out:       out,
	})
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected empty-token error, got %v", err)
	}
}
