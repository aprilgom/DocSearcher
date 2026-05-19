package scanner

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"
)

func TestWalkVisitsSupportedDocuments(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, root, "report.hwp")
	mustWriteFile(t, root, "nested/manual.PDF")
	mustWriteFile(t, root, "nested/notes.txt")
	mustWriteFile(t, root, "~$lock.hwpx")
	mustWriteFile(t, root, "draft.hwp.tmp")

	var visited []string
	if err := Walk(root, func(path string) error {
		visited = append(visited, path)
		return nil
	}); err != nil {
		t.Fatalf("Walk returned error: %v", err)
	}

	got := relativePaths(t, root, visited)
	want := []string{"nested/manual.PDF", "report.hwp"}
	sort.Strings(got)
	sort.Strings(want)

	if !slices.Equal(got, want) {
		t.Fatalf("Walk(%q) visited paths = %v, want %v", root, got, want)
	}
}

func TestWalkReturnsVisitError(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, root, "report.hwp")
	wantErr := errors.New("stop walking")

	err := Walk(root, func(path string) error {
		return wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Walk error = %v, want %v", err, wantErr)
	}
}

func mustWriteFile(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) returned error: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) returned error: %v", path, err)
	}
}

func relativePaths(t *testing.T, root string, paths []string) []string {
	t.Helper()
	relPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatalf("Rel(%q, %q) returned error: %v", root, path, err)
		}
		relPaths = append(relPaths, filepath.ToSlash(rel))
	}
	return relPaths
}
