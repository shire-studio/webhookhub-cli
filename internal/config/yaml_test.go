package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadForwards_FromCwd(t *testing.T) {
	cwd := t.TempDir()
	mustWrite(t, filepath.Join(cwd, "webhookhub.yaml"), []byte(`
forwards:
  - endpoint: stripe-test
    to: http://localhost:3000
  - endpoint: github
    to: http://localhost:3001/webhooks/github
`))

	got, source, err := LoadForwards(cwd, t.TempDir())
	if err != nil {
		t.Fatalf("LoadForwards: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
	if got[0].Slug != "stripe-test" || got[0].To != "http://localhost:3000" {
		t.Errorf("got[0]: %+v", got[0])
	}
	if got[1].Slug != "github" || got[1].To != "http://localhost:3001/webhooks/github" {
		t.Errorf("got[1]: %+v", got[1])
	}
	if source != filepath.Join(cwd, "webhookhub.yaml") {
		t.Errorf("source %q", source)
	}
}

func TestLoadForwards_FallsBackToConfigHome(t *testing.T) {
	cwd := t.TempDir()
	configHome := t.TempDir()
	mustWrite(t, filepath.Join(configHome, "webhookhub", "forwards.yaml"), []byte(`
forwards:
  - endpoint: only
    to: http://localhost:9000
`))

	got, source, err := LoadForwards(cwd, configHome)
	if err != nil {
		t.Fatalf("LoadForwards: %v", err)
	}
	if len(got) != 1 || got[0].Slug != "only" {
		t.Errorf("got %+v", got)
	}
	if source != filepath.Join(configHome, "webhookhub", "forwards.yaml") {
		t.Errorf("source %q", source)
	}
}

func TestLoadForwards_NoFileReturnsErrNoConfig(t *testing.T) {
	_, _, err := LoadForwards(t.TempDir(), t.TempDir())
	if err != ErrNoConfig {
		t.Errorf("got %v, want ErrNoConfig", err)
	}
}

func TestLoadForwards_InvalidYAMLIsError(t *testing.T) {
	cwd := t.TempDir()
	mustWrite(t, filepath.Join(cwd, "webhookhub.yaml"), []byte("forwards: not-a-list"))

	_, _, err := LoadForwards(cwd, t.TempDir())
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadForwards_EmptyForwardsIsError(t *testing.T) {
	cwd := t.TempDir()
	mustWrite(t, filepath.Join(cwd, "webhookhub.yaml"), []byte("forwards: []\n"))

	_, _, err := LoadForwards(cwd, t.TempDir())
	if err == nil || err == ErrNoConfig {
		t.Errorf("expected validation error, got %v", err)
	}
}

func mustWrite(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}
