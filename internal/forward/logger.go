package forward

import (
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shire-studio/webhookhub-cli/internal/api"
)

// Logger writes one human-readable line per replay. Mutex-guarded so
// goroutines in multi-endpoint mode don't interleave bytes.
type Logger struct {
	mu      sync.Mutex
	out     io.Writer
	verbose bool
}

// NewLogger returns a logger writing to out. If verbose is true, log
// lines also include request/response headers + body.
func NewLogger(out io.Writer, verbose bool) *Logger {
	return &Logger{out: out, verbose: verbose}
}

// LogReplay writes one log line for one replay. Format:
//
//	[15:04:05] POST /webhooks/stripe → 200 OK 42ms
//	[slug 15:04:05] POST /webhooks/stripe ✗ connection refused
//
// Verbose mode appends request headers, request body, response headers,
// response body — each on its own indented line.
func (l *Logger) LogReplay(prefix string, ev api.WebhookEvent, target *url.URL, res ReplayResult) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().Format("15:04:05")

	tag := now
	if prefix != "" {
		tag = prefix + " " + now
	}

	path := target.Path
	if path == "" {
		path = "/"
	}

	switch {
	case res.ErrorCode == "":
		fmt.Fprintf(l.out, "[%s] %s %s → %d %s %dms\n", tag, ev.Method, path, res.Status, statusText(res.Status), res.DurationMs)
	default:
		fmt.Fprintf(l.out, "[%s] %s %s ✗ %s\n", tag, ev.Method, path, errorText(res.ErrorCode))
	}

	if l.verbose {
		l.writeVerbose(ev, res)
	}
}

func (l *Logger) writeVerbose(ev api.WebhookEvent, res ReplayResult) {
	// Request headers
	if len(ev.Headers) > 0 {
		fmt.Fprintln(l.out, "    request headers:")
		keys := make([]string, 0, len(ev.Headers))
		for k := range ev.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(l.out, "      %s: %s\n", k, strings.Join(ev.Headers[k], ", "))
		}
	}
	if ev.Body != "" {
		fmt.Fprintf(l.out, "    request body: %s\n", ev.Body)
	}

	// Response
	if len(res.Headers) > 0 {
		fmt.Fprintln(l.out, "    response headers:")
		keys := make([]string, 0, len(res.Headers))
		for k := range res.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(l.out, "      %s: %s\n", k, res.Headers[k])
		}
	}
	if res.Body != "" {
		fmt.Fprintf(l.out, "    response body: %s\n", res.Body)
	}
}

func statusText(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "OK"
	case code >= 300 && code < 400:
		return "REDIRECT"
	case code == 0:
		return ""
	default:
		return "ERR"
	}
}

func errorText(code string) string {
	switch code {
	case "connection_refused":
		return "connection refused"
	case "timeout":
		return "timeout"
	case "dns_failure":
		return "dns failure"
	default:
		return code
	}
}
