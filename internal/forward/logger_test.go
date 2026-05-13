package forward

import (
	"bytes"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/shire-studio/webhookhub-cli/internal/api"
)

func TestLogger_DefaultFormat(t *testing.T) {
	out := &bytes.Buffer{}
	lg := NewLogger(out, false)

	target, _ := url.Parse("http://localhost:3000/webhooks/stripe")
	lg.LogReplay("", api.WebhookEvent{Method: "POST"}, target, ReplayResult{
		Status:     200,
		DurationMs: 42,
	})

	got := out.String()
	for _, want := range []string{"POST", "/webhooks/stripe", "200", "42ms"} {
		if !strings.Contains(got, want) {
			t.Errorf("output %q missing %q", got, want)
		}
	}
}

func TestLogger_PrefixForMultiEndpoint(t *testing.T) {
	out := &bytes.Buffer{}
	lg := NewLogger(out, false)

	target, _ := url.Parse("http://localhost:3000")
	lg.LogReplay("stripe-test", api.WebhookEvent{Method: "POST"}, target, ReplayResult{Status: 200, DurationMs: 5})

	if !strings.Contains(out.String(), "stripe-test") {
		t.Errorf("missing slug prefix in %q", out.String())
	}
}

func TestLogger_ErrorVariants(t *testing.T) {
	cases := []struct {
		errCode string
		want    string
	}{
		{"connection_refused", "connection refused"},
		{"timeout", "timeout"},
		{"dns_failure", "dns failure"},
	}
	target, _ := url.Parse("http://localhost:3000")
	for _, tc := range cases {
		out := &bytes.Buffer{}
		NewLogger(out, false).LogReplay("", api.WebhookEvent{Method: "POST"}, target, ReplayResult{
			ErrorCode: tc.errCode,
		})
		if !strings.Contains(out.String(), tc.want) {
			t.Errorf("for code %q want %q in output, got %q", tc.errCode, tc.want, out.String())
		}
	}
}

func TestLogger_VerboseIncludesHeadersAndBody(t *testing.T) {
	out := &bytes.Buffer{}
	lg := NewLogger(out, true)

	target, _ := url.Parse("http://localhost:3000")
	lg.LogReplay("", api.WebhookEvent{
		Method:  "POST",
		Headers: map[string][]string{"content-type": {"application/json"}},
		Body:    `{"x":1}`,
	}, target, ReplayResult{
		Status:     200,
		Headers:    map[string]string{"X-Local-Reply": "yes"},
		Body:       `{"ok":true}`,
		DurationMs: 5,
	})

	got := out.String()
	for _, want := range []string{"content-type", `{"x":1}`, "X-Local-Reply", `{"ok":true}`} {
		if !strings.Contains(got, want) {
			t.Errorf("verbose output missing %q\nfull: %s", want, got)
		}
	}
}

func TestLogger_ConcurrentLinesDontInterleave(t *testing.T) {
	out := &bytes.Buffer{}
	lg := NewLogger(out, false)
	target, _ := url.Parse("http://localhost:3000")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lg.LogReplay("a", api.WebhookEvent{Method: "POST"}, target, ReplayResult{Status: 200, DurationMs: 1})
		}()
	}
	wg.Wait()

	// Each line should be self-contained: every newline-terminated
	// substring should contain "a" exactly once and end cleanly.
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 50 {
		t.Errorf("got %d lines, want 50\noutput: %s", len(lines), out.String())
	}
	for i, l := range lines {
		if !strings.Contains(l, "POST") || !strings.Contains(l, "200") {
			t.Errorf("line %d corrupted: %q", i, l)
		}
	}
}
