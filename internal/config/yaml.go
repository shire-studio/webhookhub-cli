// Package config parses optional client-side config files.
//
// Currently only webhookhub.yaml (forwards mapping). Auth config (Plan 3)
// lives in internal/auth.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ErrNoConfig is returned by LoadForwards when neither candidate file exists.
var ErrNoConfig = errors.New("no webhookhub.yaml found in CWD or " +
	"$XDG_CONFIG_HOME/webhookhub/forwards.yaml")

// ForwardMapping is one entry in the forwards: list.
type ForwardMapping struct {
	Slug string // endpoint slug, e.g. "stripe-test"
	To   string // local URL to replay against, e.g. "http://localhost:3000"
}

// fileSchema is the wire shape of webhookhub.yaml — distinct from
// ForwardMapping because the on-disk field name is `endpoint:` while the
// in-memory field is `Slug` (clearer at the call site).
type fileSchema struct {
	Forwards []struct {
		Endpoint string `yaml:"endpoint"`
		To       string `yaml:"to"`
	} `yaml:"forwards"`
}

// LoadForwards looks for webhookhub.yaml in cwd, then forwards.yaml in
// configHome/webhookhub. Returns the parsed mappings, the path it loaded
// from, and any error.
//
// configHome is normally os.UserConfigDir(); callers pass it explicitly
// so tests can use t.TempDir().
func LoadForwards(cwd, configHome string) ([]ForwardMapping, string, error) {
	candidates := []string{
		filepath.Join(cwd, "webhookhub.yaml"),
		filepath.Join(configHome, "webhookhub", "forwards.yaml"),
	}

	var path string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			path = c
			break
		}
	}
	if path == "" {
		return nil, "", ErrNoConfig
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, path, fmt.Errorf("read %s: %w", path, err)
	}

	var schema fileSchema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil, path, fmt.Errorf("parse %s: %w", path, err)
	}
	if len(schema.Forwards) == 0 {
		return nil, path, fmt.Errorf("no forwards defined in %s", path)
	}

	mappings := make([]ForwardMapping, 0, len(schema.Forwards))
	for i, f := range schema.Forwards {
		if f.Endpoint == "" {
			return nil, path, fmt.Errorf("entry %d: missing endpoint", i)
		}
		if f.To == "" {
			return nil, path, fmt.Errorf("entry %d: missing to (target URL)", i)
		}
		mappings = append(mappings, ForwardMapping{Slug: f.Endpoint, To: f.To})
	}

	return mappings, path, nil
}
