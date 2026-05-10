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
