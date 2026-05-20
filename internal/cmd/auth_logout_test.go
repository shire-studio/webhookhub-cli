package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shire-studio/webhookhub-cli/internal/auth"
)

func TestAuthLogout_DeletesConfig(t *testing.T) {
	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)
	if err := store.Save(auth.Config{Token: "whk_test"}); err != nil {
		t.Fatalf("setup save: %v", err)
	}

	out := &bytes.Buffer{}
	if err := runAuthLogout(store, out); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if _, err := store.Load(); err != auth.ErrNotLoggedIn {
		t.Errorf("expected ErrNotLoggedIn after logout, got %v", err)
	}
	if !strings.Contains(out.String(), "Logged out") {
		t.Errorf("expected confirmation, got %q", out.String())
	}
}

func TestAuthLogout_IsIdempotentWhenNotLoggedIn(t *testing.T) {
	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)

	out := &bytes.Buffer{}
	if err := runAuthLogout(store, out); err != nil {
		t.Errorf("logout on empty: %v", err)
	}
	if !strings.Contains(out.String(), "Logged out") {
		t.Errorf("expected confirmation even when not logged in, got %q", out.String())
	}
}
