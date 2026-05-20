package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMe_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer whk_test" {
			t.Errorf("got auth header %q", got)
		}
		if r.URL.Path != "/api/cli/me" {
			t.Errorf("got path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":{"email":"dev@example.com","plan":"indie"}}`))
	}))
	defer server.Close()

	c := New(server.URL, "whk_test")
	me, err := c.Me(context.Background())
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if me.Email != "dev@example.com" || me.Plan != "indie" {
		t.Errorf("got %+v", me)
	}
}

func TestMe_401ReturnsErrUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer server.Close()

	c := New(server.URL, "whk_bad")
	_, err := c.Me(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("got %v, want ErrUnauthorized", err)
	}
}

func TestMe_403ReturnsErrForbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
	}))
	defer server.Close()

	c := New(server.URL, "whk_test")
	_, err := c.Me(context.Background())
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}

func TestMe_5xxReturnsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer server.Close()

	c := New(server.URL, "whk_test")
	_, err := c.Me(context.Background())
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Errorf("got %v, want error mentioning 503", err)
	}
}

func TestEndpoints_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cli/endpoints" {
			t.Errorf("got path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[
			{"slug":"stripe-test","name":"Stripe Test","url":"https://webhookhub.dev/hook/abc","active":true},
			{"slug":"github","name":"GitHub","url":"https://webhookhub.dev/hook/def","active":false}
		]}`))
	}))
	defer server.Close()

	c := New(server.URL, "whk_test")
	endpoints, err := c.Endpoints(context.Background())
	if err != nil {
		t.Fatalf("Endpoints: %v", err)
	}
	if len(endpoints) != 2 {
		t.Fatalf("got %d endpoints", len(endpoints))
	}
	if endpoints[0].Slug != "stripe-test" || !endpoints[0].Active {
		t.Errorf("first: %+v", endpoints[0])
	}
	if endpoints[1].Slug != "github" || endpoints[1].Active {
		t.Errorf("second: %+v", endpoints[1])
	}
}

func TestEndpoints_EmptyArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	c := New(server.URL, "whk_test")
	endpoints, err := c.Endpoints(context.Background())
	if err != nil {
		t.Fatalf("Endpoints: %v", err)
	}
	if len(endpoints) != 0 {
		t.Errorf("got %d, want 0", len(endpoints))
	}
}
