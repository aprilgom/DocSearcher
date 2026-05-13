package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
	if got := reloaded.WatchedPaths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("WatchedPaths() = %#v, want %#v", got, want)
	}
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
	if got := reloaded.WatchedPaths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("WatchedPaths() = %#v, want %#v", got, want)
	}
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

func TestSaveUsesDefaultConfigJSON(t *testing.T) {
	originalCurrent := Current
	t.Cleanup(func() { Current = originalCurrent })

	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(%q) returned error: %v", tempDir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	Current = &Config{WatchedPaths: []string{"/docs/default"}}
	if err := Save(); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	data, err := os.ReadFile(ConfigFile)
	if err != nil {
		t.Fatalf("ReadFile(%q) returned error: %v", ConfigFile, err)
	}
	var got Config
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if !reflect.DeepEqual(got.WatchedPaths, []string{"/docs/default"}) {
		t.Fatalf("saved watched paths = %#v, want %#v", got.WatchedPaths, []string{"/docs/default"})
	}
}
