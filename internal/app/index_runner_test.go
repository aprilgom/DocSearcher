package app

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

type recordingFileProcessor struct {
	mu        sync.Mutex
	processed []string
}

func (p *recordingFileProcessor) Process(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processed = append(p.processed, path)
	return nil
}

func (p *recordingFileProcessor) Processed() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.processed...)
}

func TestIndexRunnerRunScansSupportedDocuments(t *testing.T) {
	root := t.TempDir()
	mustWriteIndexRunnerFile(t, root, "report.hwp")
	mustWriteIndexRunnerFile(t, root, "nested/manual.PDF")
	mustWriteIndexRunnerFile(t, root, "nested/notes.txt")
	mustWriteIndexRunnerFile(t, root, "~$lock.hwpx")
	mustWriteIndexRunnerFile(t, root, "draft.hwp.tmp")

	processor := &recordingFileProcessor{}
	runner := NewIndexRunner(processor.Process)

	if err := runner.Run(root); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := relativeIndexRunnerPaths(t, root, processor.Processed())
	want := []string{"nested/manual.PDF", "report.hwp"}
	sort.Strings(got)
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("processed paths = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("processed paths = %v, want %v", got, want)
		}
	}
}

func TestIndexRunnerReportsRunningState(t *testing.T) {
	runner := NewIndexRunner(func(string) error { return nil })

	if runner.IsIndexing() {
		t.Fatal("IsIndexing() = true, want false")
	}
}

func mustWriteIndexRunnerFile(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) returned error: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) returned error: %v", path, err)
	}
}

func relativeIndexRunnerPaths(t *testing.T, root string, paths []string) []string {
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
