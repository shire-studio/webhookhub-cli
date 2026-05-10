package cmd

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shire-studio/webhookhub-cli/internal/auth"
)

func TestRunForwardSingle_HappyPath_DispatchesReplayAndPostsBack(t *testing.T) {
	// 1. Local server (the CLI replays against this)
	var localGotMethod, localGotBody string
	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		localGotMethod = r.Method
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		localGotBody = string(body[:n])
		w.WriteHeader(204)
	}))
	defer localServer.Close()

	// 2. WebhookHub fake API (SSE + post-back endpoint)
	postedBack := make(chan string, 1)
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/cli/local-responses/"):
			b := make([]byte, 4096)
			n, _ := r.Body.Read(b)
			postedBack <- string(b[:n])
			w.WriteHeader(201)
		case r.URL.Path == "/api/cli/endpoints":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"slug":"stripe-test","name":"Stripe","url":"http://api/hook/abc","active":true}]}`))
		case r.URL.Path == "/api/cli/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, _ := w.(http.Flusher)

			fmt.Fprint(w, "event: connected\ndata: {\"user\":\"u@e\",\"plan\":\"free\",\"endpoints\":[\"stripe-test\"]}\n\n")
			flusher.Flush()

			fmt.Fprintf(w, "event: webhook\ndata: {\"request_id\":99,\"endpoint_slug\":\"stripe-test\",\"method\":\"POST\",\"headers\":{\"content-type\":[\"application/json\"]},\"query_params\":{},\"body\":\"{\\\"x\\\":1}\",\"received_at\":\"2026-05-10T00:00:00Z\"}\n\n")
			flusher.Flush()

			// hold connection open until ctx cancels
			<-r.Context().Done()
		default:
			w.WriteHeader(404)
		}
	}))
	defer apiServer.Close()

	// 3. Auth store with a stored token
	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)
	_ = store.Save(auth.Config{Token: "whk_test"})

	// 4. Run the forward, cancel after we see the post-back
	ctx, cancel := context.WithCancel(context.Background())
	out := &bytes.Buffer{}
	var mu sync.Mutex

	done := make(chan error, 1)
	go func() {
		mu.Lock()
		defer mu.Unlock()
		done <- runForwardSingle(forwardOpts{
			BaseURL:   apiServer.URL,
			Store:     store,
			Slug:      "stripe-test",
			TargetURL: localServer.URL,
			Timeout:   2 * time.Second,
			Verbose:   false,
			Out:       out,
			Ctx:       ctx,
		})
	}()

	select {
	case body := <-postedBack:
		if !strings.Contains(body, "\"status\":204") {
			t.Errorf("post-back missing status:204, body=%s", body)
		}
	case <-time.After(5 * time.Second):
		cancel()
		<-done
		t.Fatal("post-back never fired")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runForwardSingle did not return after ctx cancel")
	}

	if localGotMethod != "POST" {
		t.Errorf("local got method %q", localGotMethod)
	}
	if localGotBody != `{"x":1}` {
		t.Errorf("local got body %q", localGotBody)
	}
	if !strings.Contains(out.String(), "Stripe") && !strings.Contains(out.String(), "stripe-test") {
		t.Errorf("expected log mentions endpoint, got: %s", out.String())
	}
}

func TestRunForwardSingle_UnknownSlugIsError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/cli/endpoints" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"slug":"github","name":"GH","url":"http://api/hook/x","active":true}]}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer apiServer.Close()

	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)
	_ = store.Save(auth.Config{Token: "whk_test"})

	out := &bytes.Buffer{}
	err := runForwardSingle(forwardOpts{
		BaseURL:   apiServer.URL,
		Store:     store,
		Slug:      "doesnt-exist",
		TargetURL: "http://localhost:9999",
		Timeout:   1 * time.Second,
		Out:       out,
		Ctx:       context.Background(),
	})
	if err == nil || !strings.Contains(err.Error(), "doesnt-exist") {
		t.Errorf("expected error mentioning unknown slug, got %v", err)
	}
}

func TestRunForwardSingle_TerminalUnauthorized(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/cli/endpoints":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"slug":"x","name":"X","url":"http://api/hook/x","active":true}]}`))
		case "/api/cli/stream":
			w.WriteHeader(401)
		}
	}))
	defer apiServer.Close()

	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)
	_ = store.Save(auth.Config{Token: "whk_revoked"})

	err := runForwardSingle(forwardOpts{
		BaseURL:   apiServer.URL,
		Store:     store,
		Slug:      "x",
		TargetURL: "http://localhost:9999",
		Timeout:   1 * time.Second,
		Out:       &bytes.Buffer{},
		Ctx:       context.Background(),
	})
	if err == nil || !strings.Contains(err.Error(), "rejected") {
		t.Errorf("expected token-rejected error, got %v", err)
	}
}

func TestRunForwardMulti_DispatchesByEndpointSlug(t *testing.T) {
	// Two local servers, one per slug.
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok-1"))
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok-2"))
	}))
	defer server2.Close()

	postBacks := make(chan string, 4)
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/cli/local-responses/"):
			postBacks <- r.URL.Path
			w.WriteHeader(201)
		case r.URL.Path == "/api/cli/endpoints":
			_, _ = w.Write([]byte(`{"data":[{"slug":"a","name":"A","url":"u","active":true},{"slug":"b","name":"B","url":"u","active":true}]}`))
		case r.URL.Path == "/api/cli/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, _ := w.(http.Flusher)

			fmt.Fprint(w, "event: connected\ndata: {\"user\":\"u\",\"plan\":\"free\",\"endpoints\":[\"a\",\"b\"]}\n\n")
			flusher.Flush()

			for _, slug := range []string{"a", "b"} {
				rid := 1
				if slug == "b" {
					rid = 2
				}
				fmt.Fprintf(w, "event: webhook\ndata: {\"request_id\":%d,\"endpoint_slug\":%q,\"method\":\"POST\",\"headers\":{},\"query_params\":{},\"body\":\"\",\"received_at\":\"\"}\n\n", rid, slug)
				flusher.Flush()
			}
			<-r.Context().Done()
		}
	}))
	defer apiServer.Close()

	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)
	_ = store.Save(auth.Config{Token: "whk_test"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := &bytes.Buffer{}
	done := make(chan error, 1)
	go func() {
		done <- runForwardMulti(forwardOpts{
			BaseURL: apiServer.URL,
			Store:   store,
			Mappings: map[string]string{
				"a": server1.URL,
				"b": server2.URL,
			},
			Timeout: 2 * time.Second,
			Out:     out,
			Ctx:     ctx,
		})
	}()

	got := map[string]bool{}
	timeout := time.After(5 * time.Second)
	for len(got) < 2 {
		select {
		case path := <-postBacks:
			got[path] = true
		case <-timeout:
			cancel()
			<-done
			t.Fatalf("expected 2 post-backs, got %d", len(got))
		}
	}

	cancel()
	<-done
	if !got["/api/cli/local-responses/1"] || !got["/api/cli/local-responses/2"] {
		t.Errorf("missing post-back, got %v", got)
	}
}

func TestRunForwardMulti_RejectsUnknownSlugInMappings(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/cli/endpoints" {
			_, _ = w.Write([]byte(`{"data":[{"slug":"a","name":"A","url":"u","active":true}]}`))
		}
	}))
	defer apiServer.Close()

	dir := t.TempDir()
	store := auth.NewStoreInDir(dir)
	_ = store.Save(auth.Config{Token: "whk_test"})

	err := runForwardMulti(forwardOpts{
		BaseURL: apiServer.URL,
		Store:   store,
		Mappings: map[string]string{
			"a":            "http://localhost:1",
			"doesnt-exist": "http://localhost:2",
		},
		Timeout: 1 * time.Second,
		Out:     &bytes.Buffer{},
		Ctx:     context.Background(),
	})
	if err == nil || !strings.Contains(err.Error(), "doesnt-exist") {
		t.Errorf("expected error mentioning unknown slug, got %v", err)
	}
}
