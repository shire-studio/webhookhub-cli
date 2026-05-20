package auth

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewStoreInDir(dir)

	if err := store.Save(Config{Token: "whk_test"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Token != "whk_test" {
		t.Errorf("got token %q, want whk_test", got.Token)
	}
}

func TestSave_FilePermissionsAre0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file mode bits are not enforced on Windows")
	}

	dir := t.TempDir()
	store := NewStoreInDir(dir)

	if err := store.Save(Config{Token: "whk_test"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("got mode %v, want 0600", info.Mode().Perm())
	}
}

func TestLoad_ReturnsErrNotLoggedInWhenMissing(t *testing.T) {
	dir := t.TempDir()
	store := NewStoreInDir(dir)

	_, err := store.Load()
	if err != ErrNotLoggedIn {
		t.Errorf("got %v, want ErrNotLoggedIn", err)
	}
}

func TestDelete_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	store := NewStoreInDir(dir)

	if err := store.Delete(); err != nil {
		t.Errorf("delete on empty: %v", err)
	}

	if err := store.Save(Config{Token: "whk_test"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := store.Delete(); err != nil {
		t.Errorf("delete after save: %v", err)
	}

	_, err := store.Load()
	if err != ErrNotLoggedIn {
		t.Errorf("got %v after delete, want ErrNotLoggedIn", err)
	}
}

func TestSave_CreatesParentDirectoryIfMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing", "nested")
	store := NewStoreInDir(dir)

	if err := store.Save(Config{Token: "whk_test"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "config.json")); err != nil {
		t.Errorf("expected file to exist, got %v", err)
	}
}
