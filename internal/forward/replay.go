// Package forward holds the per-event replay + log + post-back logic.
//
// Replay takes a captured WebhookEvent and a target URL, builds a fresh
// *http.Request that mimics the original (method, path-as-target,
// filtered headers, body), executes it, and returns a normalized
// ReplayResult.
package forward

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shire-studio/webhookhub-cli/internal/api"
)

// MaxBodyBytes caps the local response body we capture and post back.
// Server-side body cap is 100KB (Plan 2a) — we match it client-side so
// "truncated" is computed once.
const MaxBodyBytes = 100_000

// hopByHopHeaders are NOT forwarded — RFC 7230 §6.1.
var hopByHopHeaders = map[string]struct{}{
	"connection":          {},
	"keep-alive":          {},
	"proxy-authenticate":  {},
	"proxy-authorization": {},
	"te":                  {},
	"trailer":             {},
	"transfer-encoding":   {},
	"upgrade":             {},
	// Set by net/http from the URL/body — never carry them over.
	"host":           {},
	"content-length": {},
}

// ReplayResult is the outcome of one Replay call. Either Status > 0
// (local server replied) or ErrorCode != "" (transport failure).
type ReplayResult struct {
	Status     int
	Headers    map[string]string
	Body       string
	Truncated  bool
	DurationMs int64
	ErrorCode  string // "connection_refused" | "timeout" | "dns_failure" | ""
}

// Replay rebuilds the captured event as a fresh HTTP request to target,
// applies a per-request timeout, and returns the result.
func Replay(ctx context.Context, target *url.URL, ev api.WebhookEvent, timeout time.Duration) ReplayResult {
	start := time.Now()

	body := strings.NewReader(ev.Body)
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, ev.Method, target.String(), body)
	if err != nil {
		return ReplayResult{ErrorCode: "request_build", DurationMs: ms(start)}
	}

	for name, values := range ev.Headers {
		lowered := strings.ToLower(name)
		if _, skip := hopByHopHeaders[lowered]; skip {
			continue
		}
		// Server redacts these to literal "[REDACTED]". Don't forward.
		if len(values) == 1 && values[0] == "[REDACTED]" {
			continue
		}
		for _, v := range values {
			req.Header.Add(name, v)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ReplayResult{ErrorCode: classifyDialError(err), DurationMs: ms(start)}
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, MaxBodyBytes+1)
	bodyBytes, _ := io.ReadAll(limited)

	truncated := false
	if len(bodyBytes) > MaxBodyBytes {
		bodyBytes = bodyBytes[:MaxBodyBytes]
		truncated = true
	}

	hdrs := make(map[string]string, len(resp.Header))
	for k, vs := range resp.Header {
		if len(vs) > 0 {
			hdrs[k] = vs[0]
		}
	}

	return ReplayResult{
		Status:     resp.StatusCode,
		Headers:    hdrs,
		Body:       string(bodyBytes),
		Truncated:  truncated,
		DurationMs: ms(start),
	}
}

func ms(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}

// classifyDialError maps Go transport errors to the spec's error codes.
func classifyDialError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var nerr net.Error
	if errors.As(err, &nerr) && nerr.Timeout() {
		return "timeout"
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns_failure"
	}
	if strings.Contains(err.Error(), "connection refused") {
		return "connection_refused"
	}
	return "transport_error"
}
