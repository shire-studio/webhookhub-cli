package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shire-studio/webhookhub-cli/internal/auth"
)

func TestAuthWhoami_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer whk_stored" {
			t.Errorf("got auth %q", got)
		}
		_, _ = w.Write([]byte(`{"data":{"email":"alice@example.com","plan":"pro"}}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)
	if err := store.Save(auth.Config{Token: "whk_stored"}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	out := &bytes.Buffer{}
	if err := runAuthWhoami(server.URL, store, out); err != nil {
		t.Fatalf("whoami: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "alice@example.com") || !strings.Contains(got, "pro") {
		t.Errorf("got %q", got)
	}
}

func TestAuthWhoami_NotLoggedIn(t *testing.T) {
	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)

	out := &bytes.Buffer{}
	err := runAuthWhoami("http://example.invalid", store, out)
	if err == nil || !strings.Contains(err.Error(), "not logged in") {
		t.Errorf("expected not-logged-in error, got %v", err)
	}
}

func TestAuthWhoami_TokenRejectedByServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
	}))
	defer server.Close()

	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)
	if err := store.Save(auth.Config{Token: "whk_revoked"}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	out := &bytes.Buffer{}
	err := runAuthWhoami(server.URL, store, out)
	if err == nil || !strings.Contains(err.Error(), "rejected") {
		t.Errorf("expected token rejected, got %v", err)
	}
}
