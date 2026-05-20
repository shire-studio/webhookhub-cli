package api

// Me is the response payload of GET /api/cli/me.
type Me struct {
	Email string `json:"email"`
	Plan  string `json:"plan"`
}

// Endpoint is one element of GET /api/cli/endpoints.
type Endpoint struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	Active bool   `json:"active"`
}

// ConnectedPayload is the data field of an SSE `connected` frame.
type ConnectedPayload struct {
	User      string   `json:"user"`
	Plan      string   `json:"plan"`
	Endpoints []string `json:"endpoints"` // nil when no `?endpoints=` filter was passed
}

// WebhookEvent is the data field of an SSE `webhook` frame.
type WebhookEvent struct {
	RequestID    int64               `json:"request_id"`
	EndpointSlug string              `json:"endpoint_slug"`
	Method       string              `json:"method"`
	Headers      map[string][]string `json:"headers"`
	QueryParams  map[string]string   `json:"query_params"`
	Body         string              `json:"body"`
	ReceivedAt   string              `json:"received_at"`
}

// LocalResponseInput is the body of POST /api/cli/local-responses/{id}.
//
// Either Status > 0 (the local server replied with an HTTP status) or
// Error != "" (the request never completed). Both can be present
// together if the local server replied but the body read errored.
type LocalResponseInput struct {
	Status     int               `json:"status,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
	DurationMs int64             `json:"duration_ms,omitempty"`
	Error      string            `json:"error,omitempty"`
}
