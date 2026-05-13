package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// PostLocalResponse uploads the local server's response (status + headers
// + body + duration) for one previously-captured webhook request.
//
// Server-side this is gated by `auth:sanctum + ability:cli` and a
// per-request idempotency unique constraint. A 409 means another CLI on
// another machine got there first — for the cross-laptop "broadcast"
// model in the spec, this is normal and treated as success.
func (c *Client) PostLocalResponse(ctx context.Context, requestID int64, lr LocalResponseInput) error {
	body, err := json.Marshal(lr)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	url := fmt.Sprintf("%s/api/cli/local-responses/%d", c.baseURL, requestID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK, http.StatusConflict:
		return nil
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("post-back local response: status %d: %s", resp.StatusCode, string(respBody))
	}
}
