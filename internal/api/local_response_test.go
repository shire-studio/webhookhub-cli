package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostLocalResponse_HappyPath(t *testing.T) {
	var gotPath, gotAuth, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"data":{"id":1}}`))
	}))
	defer server.Close()

	c := New(server.URL, "whk_test")
	err := c.PostLocalResponse(context.Background(), 42, LocalResponseInput{
		Status:     200,
		Headers:    map[string]string{"content-type": "application/json"},
		Body:       "ok",
		DurationMs: 42,
	})
	if err != nil {
		t.Fatalf("PostLocalResponse: %v", err)
	}

	if gotPath != "/api/cli/local-responses/42" {
		t.Errorf("path %q", gotPath)
	}
	if gotAuth != "Bearer whk_test" {
		t.Errorf("auth %q", gotAuth)
	}

	var decoded map[string]any
	_ = json.Unmarshal([]byte(gotBody), &decoded)
	if int(decoded["status"].(float64)) != 200 {
		t.Errorf("status: %v", decoded["status"])
	}
	if int(decoded["duration_ms"].(float64)) != 42 {
		t.Errorf("duration_ms: %v", decoded["duration_ms"])
	}
}

func TestPostLocalResponse_409IsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(`{"message":"already recorded"}`))
	}))
	defer server.Close()

	c := New(server.URL, "whk_test")
	err := c.PostLocalResponse(context.Background(), 42, LocalResponseInput{Status: 200})
	if err != nil {
		t.Errorf("expected nil for 409 (idempotency), got %v", err)
	}
}

func TestPostLocalResponse_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
	}))
	defer server.Close()

	c := New(server.URL, "whk_bad")
	err := c.PostLocalResponse(context.Background(), 42, LocalResponseInput{Status: 200})
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("got %v, want ErrUnauthorized", err)
	}
}

func TestPostLocalResponse_500ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"message":"server boom"}`))
	}))
	defer server.Close()

	c := New(server.URL, "whk_test")
	err := c.PostLocalResponse(context.Background(), 42, LocalResponseInput{Status: 200})
	if err == nil {
		t.Errorf("expected error for 500, got nil")
	}
}

func TestPostLocalResponse_OmitsEmptyError(t *testing.T) {
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(201)
	}))
	defer server.Close()

	c := New(server.URL, "whk_test")
	_ = c.PostLocalResponse(context.Background(), 42, LocalResponseInput{
		Status:     200,
		DurationMs: 10,
		// no Error field
	})
	// The wire format must NOT include "error":""; use omitempty.
	if got := gotBody; got != "" && containsString(got, `"error":""`) {
		t.Errorf("expected no error field on success post, body was %q", got)
	}
}

func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	}()))
}
