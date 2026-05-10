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
