package usecase

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"sync"
	"testing"
	"time"
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

	if !slices.Equal(got, want) {
		t.Fatalf("processed paths = %v, want %v", got, want)
	}
}

func TestIndexRunnerReportsRunningState(t *testing.T) {
	runner := NewIndexRunner(func(string) error { return nil })

	if runner.IsIndexing() {
		t.Fatal("IsIndexing() = true, want false")
	}
}

func TestIndexRunnerStartPreventsDuplicateRunWhileIndexing(t *testing.T) {
	firstRoot := t.TempDir()
	secondRoot := t.TempDir()
	mustWriteIndexRunnerFile(t, firstRoot, "first.hwp")
	mustWriteIndexRunnerFile(t, secondRoot, "second.hwp")
	processor := newBlockingFileProcessor()
	runner := NewIndexRunner(processor.Process)

	runner.Start(firstRoot)
	firstPath := processor.waitStarted(t)
	if filepath.Base(firstPath) != "first.hwp" {
		t.Fatalf("first processed path = %q, want first.hwp", firstPath)
	}
	if !runner.IsIndexing() {
		t.Fatal("IsIndexing() = false while first run is blocked, want true")
	}

	runner.Start(secondRoot)
	processor.release()
	waitUntilIndexRunnerIdle(t, runner)

	got := relativeIndexRunnerPaths(t, filepath.Dir(firstRoot), processor.Processed())
	want := []string{filepath.ToSlash(filepath.Join(filepath.Base(firstRoot), "first.hwp"))}
	if !slices.Equal(got, want) {
		t.Fatalf("processed paths = %v, want %v", got, want)
	}
}

func TestIndexRunnerStartClearsRunningStateAfterCompletion(t *testing.T) {
	root := t.TempDir()
	mustWriteIndexRunnerFile(t, root, "report.pdf")
	processor := newBlockingFileProcessor()
	runner := NewIndexRunner(processor.Process)

	runner.Start(root)
	processor.waitStarted(t)
	if !runner.IsIndexing() {
		t.Fatal("IsIndexing() = false while run is blocked, want true")
	}

	processor.release()
	waitUntilIndexRunnerIdle(t, runner)

	if runner.IsIndexing() {
		t.Fatal("IsIndexing() = true after completion, want false")
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

type blockingFileProcessor struct {
	recordingFileProcessor
	started     chan string
	releaseOnce sync.Once
	releaseCh   chan struct{}
}

func newBlockingFileProcessor() *blockingFileProcessor {
	return &blockingFileProcessor{
		started:   make(chan string, 1),
		releaseCh: make(chan struct{}),
	}
}

func (p *blockingFileProcessor) Process(path string) error {
	p.started <- path
	<-p.releaseCh
	return p.recordingFileProcessor.Process(path)
}

func (p *blockingFileProcessor) waitStarted(t *testing.T) string {
	t.Helper()
	select {
	case path := <-p.started:
		return path
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for processor to start")
		return ""
	}
}

func (p *blockingFileProcessor) release() {
	p.releaseOnce.Do(func() {
		close(p.releaseCh)
	})
}

func waitUntilIndexRunnerIdle(t *testing.T, runner *IndexRunner) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for runner.IsIndexing() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for runner to become idle")
		}
		runtime.Gosched()
	}
}
