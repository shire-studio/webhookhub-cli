// Package auth manages the on-disk auth config for the webhookhub CLI.
package auth

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrNotLoggedIn is returned by Load when no config file exists.
var ErrNotLoggedIn = errors.New("not logged in (run `webhookhub auth login`)")

// Config is the persisted shape of ~/.config/webhookhub/config.json.
type Config struct {
	Token string `json:"token"`
}

// Store reads, writes, and deletes the auth config in a fixed directory.
type Store struct {
	dir string
}

// NewStore returns a Store rooted at $XDG_CONFIG_HOME/webhookhub (or the OS
// equivalent on macOS/Windows via os.UserConfigDir).
func NewStore() (*Store, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return &Store{dir: filepath.Join(base, "webhookhub")}, nil
}

// NewStoreInDir is for tests — it bypasses os.UserConfigDir and uses the
// caller-supplied directory.
func NewStoreInDir(dir string) *Store {
	return &Store{dir: dir}
}

func (s *Store) path() string {
	return filepath.Join(s.dir, "config.json")
}

// Save writes the config with mode 0600. Creates the parent directory if
// missing.
func (s *Store) Save(cfg Config) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path(), data, 0o600)
}

// Load reads the config. Returns ErrNotLoggedIn if the file is missing.
func (s *Store) Load() (Config, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Config{}, ErrNotLoggedIn
		}
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Delete removes the config file. Idempotent — does NOT return an error if
// the file doesn't exist.
func (s *Store) Delete() error {
	err := os.Remove(s.path())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
