package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shire-studio/webhookhub-cli/internal/auth"
)

func TestEndpoints_PrintsTabularList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[
			{"slug":"stripe-test","name":"Stripe Test","url":"https://webhookhub.dev/hook/abc","active":true},
			{"slug":"gh","name":"GitHub","url":"https://webhookhub.dev/hook/def","active":false}
		]}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)
	_ = store.Save(auth.Config{Token: "whk_test"})

	out := &bytes.Buffer{}
	if err := runEndpoints(server.URL, store, out); err != nil {
		t.Fatalf("endpoints: %v", err)
	}

	got := out.String()
	for _, want := range []string{"stripe-test", "Stripe Test", "https://webhookhub.dev/hook/abc", "gh", "GitHub", "(inactive)"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\nfull: %s", want, got)
		}
	}
}

func TestEndpoints_EmptyState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)
	_ = store.Save(auth.Config{Token: "whk_test"})

	out := &bytes.Buffer{}
	if err := runEndpoints(server.URL, store, out); err != nil {
		t.Fatalf("endpoints: %v", err)
	}
	if !strings.Contains(out.String(), "No endpoints") {
		t.Errorf("expected empty-state message, got %q", out.String())
	}
}

func TestEndpoints_NotLoggedIn(t *testing.T) {
	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)

	out := &bytes.Buffer{}
	err := runEndpoints("http://example.invalid", store, out)
	if err == nil || !strings.Contains(err.Error(), "not logged in") {
		t.Errorf("expected not-logged-in, got %v", err)
	}
}
