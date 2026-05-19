package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"hwp-searcher/internal/domain"
)

func TestStorePersistsWatchedPathsToInjectedPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "custom-config.json")
	store := NewStore(configPath)

	if err := store.AddPath(domain.WatchedPath("/docs/alpha")); err != nil {
		t.Fatalf("AddPath returned error: %v", err)
	}
	if err := store.AddPath(domain.WatchedPath("/docs/beta")); err != nil {
		t.Fatalf("AddPath returned error: %v", err)
	}
	if err := store.AddPath(domain.WatchedPath("/docs/alpha")); err != nil {
		t.Fatalf("AddPath duplicate returned error: %v", err)
	}

	reloaded := NewStore(configPath)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	want := []domain.WatchedPath{"/docs/alpha", "/docs/beta"}
	assertWatchedPaths(t, reloaded.WatchedPaths(), want, "after reloading persisted watched paths")
}

func TestStoreRemovePathPersistsToInjectedPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "custom-config.json")
	store := NewStore(configPath)
	for _, path := range []domain.WatchedPath{"/docs/alpha", "/docs/beta"} {
		if err := store.AddPath(path); err != nil {
			t.Fatalf("AddPath(%q) returned error: %v", path, err)
		}
	}

	if err := store.RemovePath(domain.WatchedPath("/docs/alpha")); err != nil {
		t.Fatalf("RemovePath returned error: %v", err)
	}
	if err := store.RemovePath(domain.WatchedPath("/docs/missing")); err != nil {
		t.Fatalf("RemovePath missing path returned error: %v", err)
	}

	reloaded := NewStore(configPath)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	want := []domain.WatchedPath{"/docs/beta"}
	assertWatchedPaths(t, reloaded.WatchedPaths(), want, "after removing watched path")
}

func TestStoreLoadMissingInjectedPathUsesEmptyConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing-config.json")
	store := NewStore(configPath)

	if err := store.Load(); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got := store.WatchedPaths(); len(got) != 0 {
		t.Fatalf("WatchedPaths() = %#v, want empty", got)
	}
}

func TestStoreSaveWritesWatchedPathsToInjectedPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "custom-config.json")
	store := Store{
		path:    configPath,
		current: &Config{WatchedPaths: []string{"/docs/default"}},
	}

	if err := store.save(); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) returned error: %v", configPath, err)
	}
	var got Config
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	assertStrings(t, got.WatchedPaths, []string{"/docs/default"}, "saved watched paths")
}

func TestStoreLoadRejectsInvalidJSON(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "invalid-config.json")
	if err := os.WriteFile(configPath, []byte(`{"watched_paths": [`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	store := NewStore(configPath)

	if err := store.Load(); err == nil {
		t.Fatal("Load returned nil, want invalid JSON error")
	}
}

func TestStoreLoadReturnsFilesystemError(t *testing.T) {
	configPath := t.TempDir()
	store := NewStore(configPath)

	if err := store.Load(); err == nil {
		t.Fatal("Load returned nil for directory path, want filesystem error")
	}
}

func TestStoreSaveReturnsFilesystemError(t *testing.T) {
	configPath := t.TempDir()
	store := Store{
		path:    configPath,
		current: &Config{WatchedPaths: []string{"/docs/default"}},
	}

	if err := store.save(); err == nil {
		t.Fatal("save returned nil for directory path, want filesystem error")
	}
}

func assertWatchedPaths(t *testing.T, got []domain.WatchedPath, want []domain.WatchedPath, scenario string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d; got %#v, want %#v", scenario, len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q; got %#v, want %#v", scenario, i, got[i], want[i], got, want)
		}
	}
}

func assertStrings(t *testing.T, got []string, want []string, scenario string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d; got %#v, want %#v", scenario, len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q; got %#v, want %#v", scenario, i, got[i], want[i], got, want)
		}
	}
}
