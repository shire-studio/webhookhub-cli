package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStream_DispatchesConnectedAndWebhookFrames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("endpoints"); got != "stripe-test,github" {
			t.Errorf("got endpoints query %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer whk_test" {
			t.Errorf("got auth %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)

		fmt.Fprint(w, "event: connected\ndata: {\"user\":\"u@e\",\"plan\":\"indie\",\"endpoints\":[\"stripe-test\",\"github\"]}\n\n")
		flusher.Flush()

		fmt.Fprint(w, "event: webhook\ndata: {\"request_id\":42,\"endpoint_slug\":\"stripe-test\",\"method\":\"POST\",\"headers\":{\"content-type\":[\"application/json\"]},\"query_params\":{\"foo\":\"bar\"},\"body\":\"{\\\"x\\\":1}\",\"received_at\":\"2026-05-10T12:00:00Z\"}\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	var (
		mu           sync.Mutex
		gotConnected *ConnectedPayload
		gotWebhook   *WebhookEvent
		gotPing      int
	)
	c := New(server.URL, "whk_test")
	err := c.Stream(context.Background(), []string{"stripe-test", "github"}, StreamHandlers{
		OnConnected: func(cp ConnectedPayload) { mu.Lock(); defer mu.Unlock(); gotConnected = &cp },
		OnWebhook:   func(w WebhookEvent) { mu.Lock(); defer mu.Unlock(); gotWebhook = &w },
		OnPing:      func() { mu.Lock(); defer mu.Unlock(); gotPing++ },
	})
	// httptest server closes after the handler returns, which surfaces as io.EOF on the read.
	if err != nil && !errors.Is(err, io.EOF) && !strings.Contains(err.Error(), "EOF") && !strings.Contains(err.Error(), "stream closed") {
		t.Fatalf("Stream: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotConnected == nil || gotConnected.User != "u@e" || gotConnected.Plan != "indie" {
		t.Errorf("connected: %+v", gotConnected)
	}
	if gotWebhook == nil || gotWebhook.RequestID != 42 || gotWebhook.EndpointSlug != "stripe-test" || gotWebhook.Method != "POST" {
		t.Errorf("webhook: %+v", gotWebhook)
	}
	if gotWebhook != nil && gotWebhook.Body != `{"x":1}` {
		t.Errorf("body: %q", gotWebhook.Body)
	}
}

func TestStream_DispatchesPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, "event: ping\ndata: {}\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	pings := 0
	c := New(server.URL, "whk_test")
	_ = c.Stream(context.Background(), nil, StreamHandlers{
		OnPing: func() { pings++ },
	})
	if pings != 1 {
		t.Errorf("got %d pings, want 1", pings)
	}
}

func TestStream_OmitsEndpointsQueryWhenNoneProvided(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "text/event-stream")
	}))
	defer server.Close()

	c := New(server.URL, "whk_test")
	_ = c.Stream(context.Background(), nil, StreamHandlers{})
	if gotQuery != "" {
		t.Errorf("expected no query string, got %q", gotQuery)
	}
}

func TestStream_401ReturnsErrUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
	}))
	defer server.Close()

	c := New(server.URL, "whk_bad")
	err := c.Stream(context.Background(), nil, StreamHandlers{})
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("got %v, want ErrUnauthorized", err)
	}
}

func TestStream_403ReturnsErrForbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(403)
	}))
	defer server.Close()

	c := New(server.URL, "whk_test")
	err := c.Stream(context.Background(), nil, StreamHandlers{})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}

func TestStream_ContextCancellationStopsStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, "event: connected\ndata: {\"user\":\"u\",\"plan\":\"free\",\"endpoints\":null}\n\n")
		flusher.Flush()
		// Hold the connection open; rely on context cancellation to unblock.
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	connected := make(chan struct{})
	c := New(server.URL, "whk_test")

	done := make(chan error, 1)
	go func() {
		done <- c.Stream(ctx, nil, StreamHandlers{
			OnConnected: func(ConnectedPayload) { close(connected) },
		})
	}()

	select {
	case <-connected:
	case <-time.After(2 * time.Second):
		t.Fatal("connected callback never fired")
	}

	cancel()

	select {
	case err := <-done:
		// Either context.Canceled or a wrapped variant. Either way, Stream returned.
		if err == nil {
			t.Errorf("expected non-nil error after cancel, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Stream did not return after ctx cancel")
	}
}
