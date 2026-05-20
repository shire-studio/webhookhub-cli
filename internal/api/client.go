// Package api wraps the WebhookHub CLI HTTP API behind a Bearer-authenticated
// client. The client is goroutine-safe; the only mutable state is the
// embedded *http.Client which net/http guarantees safe.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrUnauthorized is returned for HTTP 401 (token rejected).
var ErrUnauthorized = errors.New("unauthorized — token rejected")

// ErrForbidden is returned for HTTP 403 (token lacks the cli ability).
var ErrForbidden = errors.New("forbidden — token does not have the 'cli' ability")

// Client is the WebhookHub CLI HTTP client.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New returns a Client with sensible defaults (10s timeout). For long-lived
// requests like the SSE stream (Plan 4), use NewWithHTTPClient with a
// per-request-shaped *http.Client.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Me calls GET /api/cli/me.
func (c *Client) Me(ctx context.Context) (*Me, error) {
	var wrapper struct {
		Data Me `json:"data"`
	}
	if err := c.get(ctx, "/api/cli/me", &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Data, nil
}

// Endpoints calls GET /api/cli/endpoints.
func (c *Client) Endpoints(ctx context.Context) ([]Endpoint, error) {
	var wrapper struct {
		Data []Endpoint `json:"data"`
	}
	if err := c.get(ctx, "/api/cli/endpoints", &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Data, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return json.NewDecoder(resp.Body).Decode(out)
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	default:
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d from %s: %s", resp.StatusCode, path, string(body))
	}
}
