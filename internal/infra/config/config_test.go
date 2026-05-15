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

func TestStoreLoadReplacesPreviouslyLoadedConfig(t *testing.T) {
	tempDir := t.TempDir()
	rootPath := filepath.Join(tempDir, "documents")
	if err := os.Mkdir(rootPath, 0755); err != nil {
		t.Fatalf("Mkdir(%q) returned error: %v", rootPath, err)
	}
	configPath := filepath.Join(tempDir, "config.json")
	firstConfig := []byte(`{
  "watched_paths": ["/legacy/docs"],
  "document_roots": [
    {"id":"documents","server_path":"` + filepath.ToSlash(rootPath) + `","smb_host":"docserver","smb_share":"documents"}
  ]
}`)
	if err := os.WriteFile(configPath, firstConfig, 0644); err != nil {
		t.Fatalf("WriteFile first config returned error: %v", err)
	}
	store := NewStore(configPath)
	if err := store.Load(); err != nil {
		t.Fatalf("Load first config returned error: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(`{}`), 0644); err != nil {
		t.Fatalf("WriteFile empty config returned error: %v", err)
	}
	if err := store.Load(); err != nil {
		t.Fatalf("Load empty config returned error: %v", err)
	}

	if got := store.WatchedPaths(); len(got) != 0 {
		t.Fatalf("WatchedPaths() after reload = %#v, want empty", got)
	}
	if got := store.DocumentRoots(); len(got) != 0 {
		t.Fatalf("DocumentRoots() after reload = %#v, want empty", got)
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

func TestStoreLoadDocumentRootsPreservesWatchedPaths(t *testing.T) {
	tempDir := t.TempDir()
	rootPath := filepath.Join(tempDir, "documents")
	if err := os.Mkdir(rootPath, 0755); err != nil {
		t.Fatalf("Mkdir(%q) returned error: %v", rootPath, err)
	}
	configPath := filepath.Join(tempDir, "config.json")
	data := []byte(`{
  "watched_paths": ["/legacy/docs"],
  "document_roots": [
    {
      "id": "documents",
      "name": "문서 공유",
      "server_path": "` + filepath.ToSlash(rootPath) + `",
      "smb_host": " docserver ",
      "smb_share": " documents ",
      "smb_aliases": [
        { "host": "dfs-docs", "share": " documents " }
      ]
    }
  ]
}`)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	store := NewStore(configPath)
	if err := store.Load(); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got := store.WatchedPaths(); !reflect.DeepEqual(got, []domain.WatchedPath{"/legacy/docs"}) {
		t.Fatalf("WatchedPaths() = %#v, want legacy path", got)
	}
	roots := store.DocumentRoots()
	if len(roots) != 1 {
		t.Fatalf("DocumentRoots() length = %d, want 1", len(roots))
	}
	root := roots[0]
	canonicalRootPath, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) returned error: %v", rootPath, err)
	}
	if root.ID != "documents" || root.Name != "문서 공유" || root.ServerPath != filepath.Clean(canonicalRootPath) {
		t.Fatalf("DocumentRoots()[0] = %#v, want loaded root with cleaned server path", root)
	}
	if root.SMB.Host != "docserver" || root.SMB.Share != "documents" {
		t.Fatalf("DocumentRoots()[0].SMB = %#v, want trimmed SMB metadata", root.SMB)
	}
	if len(root.SMBAliases) != 1 || root.SMBAliases[0].Host != "dfs-docs" || root.SMBAliases[0].Share != "documents" {
		t.Fatalf("DocumentRoots()[0].SMBAliases = %#v, want trimmed alias", root.SMBAliases)
	}
}

func TestStoreLoadRejectsInvalidDocumentRoots(t *testing.T) {
	tempDir := t.TempDir()
	rootPath := filepath.Join(tempDir, "documents")
	if err := os.Mkdir(rootPath, 0755); err != nil {
		t.Fatalf("Mkdir(%q) returned error: %v", rootPath, err)
	}
	symlinkPath := filepath.Join(tempDir, "documents-link")
	if err := os.Symlink(rootPath, symlinkPath); err != nil {
		t.Fatalf("Symlink(%q, %q) returned error: %v", rootPath, symlinkPath, err)
	}
	brokenSymlinkPath := filepath.Join(tempDir, "broken-link")
	if err := os.Symlink(filepath.Join(tempDir, "missing"), brokenSymlinkPath); err != nil {
		t.Fatalf("Symlink broken target returned error: %v", err)
	}

	tests := map[string]string{
		"duplicate root id": `[
		  {"id":"documents","server_path":"` + filepath.ToSlash(rootPath) + `","smb_host":"docserver","smb_share":"documents"},
		  {"id":"documents","server_path":"` + filepath.ToSlash(filepath.Join(tempDir, "other")) + `","smb_host":"docserver","smb_share":"other"}
		]`,
		"relative server path": `[
		  {"id":"documents","server_path":"relative/docs","smb_host":"docserver","smb_share":"documents"}
		]`,
		"duplicate server path": `[
		  {"id":"documents","server_path":"` + filepath.ToSlash(rootPath) + `","smb_host":"docserver","smb_share":"documents"},
		  {"id":"archive","server_path":"` + filepath.ToSlash(rootPath) + `/.","smb_host":"docserver","smb_share":"archive"}
		]`,
		"duplicate canonical symlink server path": `[
		  {"id":"documents","server_path":"` + filepath.ToSlash(rootPath) + `","smb_host":"docserver","smb_share":"documents"},
		  {"id":"archive","server_path":"` + filepath.ToSlash(symlinkPath) + `","smb_host":"docserver","smb_share":"archive"}
		]`,
		"unresolved symlink server path": `[
		  {"id":"documents","server_path":"` + filepath.ToSlash(brokenSymlinkPath) + `","smb_host":"docserver","smb_share":"documents"}
		]`,
		"invalid smb alias": `[
		  {"id":"documents","server_path":"` + filepath.ToSlash(rootPath) + `","smb_host":"docserver","smb_share":"documents","smb_aliases":[{"host":"bad:host","share":"documents"}]}
		]`,
	}

	for name, rootsJSON := range tests {
		t.Run(name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(configPath, []byte(`{"document_roots":`+rootsJSON+`}`), 0644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			store := NewStore(configPath)
			if err := store.Load(); err == nil {
				t.Fatal("Load returned nil, want error")
			}
		})
	}
}
