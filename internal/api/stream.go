package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// StreamHandlers receives one callback per SSE frame.
//
// All callbacks are invoked from the same goroutine that called Stream.
// Don't block for long inside a callback — you'll back up the SSE read
// loop. Treat OnWebhook as fast-path; if you need to do slow work,
// dispatch to a goroutine inside the callback.
type StreamHandlers struct {
	OnConnected func(ConnectedPayload)
	OnWebhook   func(WebhookEvent)
	OnPing      func()
}

// Stream opens GET /api/cli/stream and dispatches frames to handlers
// until ctx is cancelled, the server closes the connection, or an error
// occurs. Returns ErrUnauthorized on 401, ErrForbidden on 403, or a
// wrapped error otherwise. context.Canceled / context.DeadlineExceeded
// are returned as-is.
//
// The connection has no global timeout — Stream is meant to be called
// inside a reconnect loop that handles long-lived flakiness.
func (c *Client) Stream(ctx context.Context, endpoints []string, h StreamHandlers) error {
	url := c.baseURL + "/api/cli/stream"
	if len(endpoints) > 0 {
		url += "?endpoints=" + strings.Join(endpoints, ",")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// The default Client has a 10s timeout — too short for SSE. Use a
	// dedicated http.Client without a global timeout. Per-request
	// cancellation is via ctx.
	streamClient := &http.Client{Timeout: 0}
	resp, err := streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// fallthrough
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	default:
		return fmt.Errorf("unexpected status %d from /api/cli/stream", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	// Headers/body fields can grow large. 10MB is well above the server's
	// 1MB body cap; 64KB default is too small for some real webhooks.
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	var event, data string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line dispatches the buffered frame.
			if event != "" {
				dispatch(event, data, h)
			}
			event, data = "", ""
			continue
		}

		switch {
		case strings.HasPrefix(line, "event: "):
			event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			data = strings.TrimPrefix(line, "data: ")
		}
		// Other lines (including SSE comment lines starting with `:`,
		// e.g. `:keep-alive`) are ignored per the SSE spec.
	}

	if err := scanner.Err(); err != nil {
		// ctx cancellation surfaces as a wrapped error from the read.
		// Pass through unmodified so callers can distinguish.
		return err
	}

	// Server closed cleanly (EOF). Surface a hint so callers know to reconnect.
	return fmt.Errorf("stream closed")
}

func dispatch(event, data string, h StreamHandlers) {
	switch event {
	case "connected":
		var p ConnectedPayload
		if err := json.Unmarshal([]byte(data), &p); err == nil && h.OnConnected != nil {
			h.OnConnected(p)
		}
	case "webhook":
		var w WebhookEvent
		if err := json.Unmarshal([]byte(data), &w); err == nil && h.OnWebhook != nil {
			h.OnWebhook(w)
		}
	case "ping":
		if h.OnPing != nil {
			h.OnPing()
		}
	}
}
