package forward

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/shire-studio/webhookhub-cli/internal/api"
)

func TestReplay_HappyPath(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotCT, gotXSig, gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		gotXSig = r.Header.Get("Stripe-Signature")
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("X-Local-Reply", "yes")
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"created":true}`))
	}))
	defer server.Close()

	target, _ := url.Parse(server.URL + "/webhooks/stripe")
	res := Replay(context.Background(), target, api.WebhookEvent{
		Method: "POST",
		Headers: map[string][]string{
			"content-type":     {"application/json"},
			"stripe-signature": {"sig_xxx"},
			"authorization":    {"[REDACTED]"}, // server redacted it
			"connection":       {"keep-alive"}, // hop-by-hop
			"host":             {"webhookhub.dev"},
			"content-length":   {"99"},
		},
		Body: `{"x":1}`,
	}, 5*time.Second)

	if res.ErrorCode != "" {
		t.Fatalf("unexpected error: %s", res.ErrorCode)
	}
	if res.Status != 201 {
		t.Errorf("status %d", res.Status)
	}
	if res.DurationMs < 0 {
		t.Errorf("duration negative: %d", res.DurationMs)
	}
	if res.Body != `{"created":true}` {
		t.Errorf("body %q", res.Body)
	}

	if gotMethod != "POST" || gotPath != "/webhooks/stripe" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	if gotBody != `{"x":1}` {
		t.Errorf("body forwarded as %q", gotBody)
	}
	if gotCT != "application/json" {
		t.Errorf("content-type %q", gotCT)
	}
	if gotXSig != "sig_xxx" {
		t.Errorf("stripe-signature %q", gotXSig)
	}
	if gotAuth != "" {
		t.Errorf("expected redacted authorization to be dropped, got %q", gotAuth)
	}
	if res.Headers["X-Local-Reply"] != "yes" {
		t.Errorf("response headers: %v", res.Headers)
	}
}

func TestReplay_DropsHopByHopHeaders(t *testing.T) {
	var gotConn, gotTE, gotUpgrade string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotConn = r.Header.Get("Connection")
		gotTE = r.Header.Get("TE")
		gotUpgrade = r.Header.Get("Upgrade")
	}))
	defer server.Close()

	target, _ := url.Parse(server.URL)
	_ = Replay(context.Background(), target, api.WebhookEvent{
		Method: "GET",
		Headers: map[string][]string{
			"connection": {"keep-alive"},
			"te":         {"gzip"},
			"upgrade":    {"websocket"},
		},
	}, 5*time.Second)

	// net/http may set its own Connection header; the assertion is that
	// our forwarded values aren't visible.
	if gotConn == "keep-alive" {
		t.Errorf("hop-by-hop Connection leaked")
	}
	if gotTE != "" {
		t.Errorf("hop-by-hop TE leaked: %q", gotTE)
	}
	if gotUpgrade != "" {
		t.Errorf("hop-by-hop Upgrade leaked: %q", gotUpgrade)
	}
}

func TestReplay_ConnectionRefused(t *testing.T) {
	target, _ := url.Parse("http://127.0.0.1:1") // port 1 — nothing listening
	res := Replay(context.Background(), target, api.WebhookEvent{
		Method: "POST",
	}, 2*time.Second)

	if res.ErrorCode != "connection_refused" {
		t.Errorf("got error code %q, want connection_refused", res.ErrorCode)
	}
	if res.Status != 0 {
		t.Errorf("expected status 0 on dial error, got %d", res.Status)
	}
}

func TestReplay_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hold the response open until the request context cancels.
		<-r.Context().Done()
	}))
	defer server.Close()

	target, _ := url.Parse(server.URL)
	res := Replay(context.Background(), target, api.WebhookEvent{
		Method: "GET",
	}, 100*time.Millisecond)

	if res.ErrorCode != "timeout" {
		t.Errorf("got error code %q, want timeout", res.ErrorCode)
	}
}

func TestReplay_DnsFailure(t *testing.T) {
	target, _ := url.Parse("http://no-such-host-3a8c2f.example.invalid/")
	res := Replay(context.Background(), target, api.WebhookEvent{
		Method: "GET",
	}, 2*time.Second)

	if res.ErrorCode != "dns_failure" {
		t.Errorf("got error code %q, want dns_failure", res.ErrorCode)
	}
}

func TestReplay_BodyTruncatedAtCap(t *testing.T) {
	huge := strings.Repeat("A", 200_000) // 200KB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(huge))
	}))
	defer server.Close()

	target, _ := url.Parse(server.URL)
	res := Replay(context.Background(), target, api.WebhookEvent{Method: "GET"}, 5*time.Second)

	if len(res.Body) != 100_000 {
		t.Errorf("expected body cap at 100,000 bytes, got %d", len(res.Body))
	}
	if !res.Truncated {
		t.Errorf("expected Truncated=true")
	}
}

// helper: ensure 127.0.0.1 dialer behaves the same regardless of test env.
func init() {
	http.DefaultTransport.(*http.Transport).DialContext = (&net.Dialer{
		Timeout: 1 * time.Second,
	}).DialContext
}
