package indexer

import (
	"hwp-searcher/internal/domain"
	"testing"
)

type fakeFileIndexer struct {
	indexed []string
	removed []string
}

func (f *fakeFileIndexer) IndexFile(path string) error {
	f.indexed = append(f.indexed, path)
	return nil
}

func (f *fakeFileIndexer) RemoveFile(path string) error {
	f.removed = append(f.removed, path)
	return nil
}

func TestIsSupportedDocumentFile(t *testing.T) {
	tests := map[string]bool{
		"report.hwp":    true,
		"report.hwpx":   true,
		"manual.pdf":    true,
		"notes.txt":     false,
		"draft.hwp.tmp": false,
		"~$lock.pdf":    false,
	}

	for path, want := range tests {
		if got := IsSupportedDocumentFile(path); got != want {
			t.Fatalf("IsSupportedDocumentFile(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestNormalizeNoSpaceContent(t *testing.T) {
	content := "한 글\nA\tB\r C"
	want := "한글ABC"

	if got := NormalizeNoSpaceContent(content); got != want {
		t.Fatalf("NormalizeNoSpaceContent() = %q, want %q", got, want)
	}
}

func TestRunnerIndexesAndRemovesWithInjectedIndexer(t *testing.T) {
	indexer := &fakeFileIndexer{}
	runner := NewRunner(indexer)

	runner.IndexFile("report.hwp")
	runner.RemoveFile("report.hwp")

	if got := len(indexer.indexed); got != 1 {
		t.Fatalf("indexed count = %d, want 1", got)
	}
	if domain.DocumentID(indexer.indexed[0]) != "report.hwp" {
		t.Fatalf("indexed path = %q, want report.hwp", indexer.indexed[0])
	}
	if got := len(indexer.removed); got != 1 {
		t.Fatalf("removed count = %d, want 1", got)
	}
	if domain.DocumentID(indexer.removed[0]) != "report.hwp" {
		t.Fatalf("removed path = %q, want report.hwp", indexer.removed[0])
	}
}
