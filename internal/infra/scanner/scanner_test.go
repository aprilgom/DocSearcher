package scanner

import (
	"os"
	"path/filepath"
	"reflect"
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

	if len(got) != len(want) {
		t.Fatalf("visited paths = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("visited paths = %v, want %v", got, want)
		}
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

func TestWalkSkipsSymlinkEntries(t *testing.T) {
	// given
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "real.hwp"), []byte("real"), 0o644); err != nil {
		t.Fatalf("WriteFile real: %v", err)
	}
	targetDir := filepath.Join(root, "target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatalf("Mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "inside.hwp"), []byte("inside"), 0o644); err != nil {
		t.Fatalf("WriteFile inside: %v", err)
	}
	if err := os.Symlink(filepath.Join(root, "real.hwp"), filepath.Join(root, "linked.hwp")); err != nil {
		t.Skipf("Symlink unsupported: %v", err)
	}
	if err := os.Symlink(targetDir, filepath.Join(root, "linked-dir")); err != nil {
		t.Skipf("directory Symlink unsupported: %v", err)
	}

	var got []string
	// when
	err := Walk(root, func(path string) error {
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			t.Fatalf("Rel: %v", relErr)
		}
		got = append(got, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	sort.Strings(got)

	// then
	want := []string{"real.hwp", "target/inside.hwp"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("visited = %#v, want %#v", got, want)
	}
}
