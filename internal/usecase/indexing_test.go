package usecase

import (
	"hwp-searcher/internal/domain"
	"os"
	"path/filepath"
	"testing"
)

func TestIndexerIndexesExtractedDocument(t *testing.T) {
	extractor := &fakeExtractor{text: "한 글"}
	index := &fakeIndex{}
	indexer := NewIndexer(extractor, index)

	err := indexer.IndexFile("report.hwp")

	if err != nil {
		t.Fatalf("IndexFile returned error: %v", err)
	}
	if extractor.path != "report.hwp" {
		t.Fatalf("extractor path = %q, want %q", extractor.path, "report.hwp")
	}
	if index.indexed == nil {
		t.Fatal("IndexDocument was not called")
	}
	if index.indexed.ContentNoSpace != "한글" {
		t.Fatalf("ContentNoSpace = %q, want %q", index.indexed.ContentNoSpace, "한글")
	}
}

func TestIndexerIndexesDocumentWithLogicalIdentityFromMostSpecificRoot(t *testing.T) {
	// given
	root := t.TempDir()
	childRoot := filepath.Join(root, "shared")
	filePath := filepath.Join(childRoot, "2026", "sample.hwp")
	extractor := &fakeExtractor{text: "한 글"}
	index := &fakeIndex{}
	indexer := NewIndexer(extractor, index, []domain.DocumentRoot{
		{ID: "documents", ServerPath: root},
		{ID: "shared", ServerPath: childRoot},
	})

	// when
	err := indexer.IndexFile(filePath)

	// then
	if err != nil {
		t.Fatalf("IndexFile returned error: %v", err)
	}
	if index.indexed == nil {
		t.Fatal("IndexDocument was not called")
	}
	if index.indexed.ID != "shared:2026/sample.hwp" {
		t.Fatalf("ID = %q, want logical document ID", index.indexed.ID)
	}
	if index.indexed.RootID != "shared" {
		t.Fatalf("RootID = %q, want shared", index.indexed.RootID)
	}
	if index.indexed.RelativePath != "2026/sample.hwp" {
		t.Fatalf("RelativePath = %q, want slash-normalized relative path", index.indexed.RelativePath)
	}
	if index.indexed.Path != filePath {
		t.Fatalf("Path = %q, want server path", index.indexed.Path)
	}
	if index.indexed.ServerPath != filePath {
		t.Fatalf("ServerPath = %q, want diagnostic server path", index.indexed.ServerPath)
	}
}

func TestIndexerSkipsSymlinkFile(t *testing.T) {
	// given
	root := t.TempDir()
	realPath := filepath.Join(root, "real.hwp")
	if err := os.WriteFile(realPath, []byte("real"), 0o644); err != nil {
		t.Fatalf("WriteFile real: %v", err)
	}
	linkPath := filepath.Join(root, "linked.hwp")
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Skipf("Symlink unsupported: %v", err)
	}
	extractor := &fakeExtractor{text: "한 글"}
	index := &fakeIndex{}
	indexer := NewIndexer(extractor, index, []domain.DocumentRoot{{ID: "documents", ServerPath: root}})

	// when
	err := indexer.IndexFile(linkPath)

	// then
	if err != nil {
		t.Fatalf("IndexFile returned error: %v", err)
	}
	if extractor.path != "" {
		t.Fatalf("extractor path = %q, want no extraction", extractor.path)
	}
	if index.indexed != nil {
		t.Fatalf("IndexDocument was called for symlink: %#v", index.indexed)
	}
}

func TestIndexerRemovesDocument(t *testing.T) {
	index := &fakeIndex{}
	indexer := NewIndexer(nil, index)

	err := indexer.RemoveFile("report.pdf")

	if err != nil {
		t.Fatalf("RemoveFile returned error: %v", err)
	}
	if index.deleted != domain.DocumentID("report.pdf") {
		t.Fatalf("deleted = %q, want %q", index.deleted, "report.pdf")
	}
}

func TestIndexerRemovesLogicalDocumentFromConfiguredRoot(t *testing.T) {
	// given
	root := t.TempDir()
	filePath := filepath.Join(root, "nested", "report.pdf")
	index := &fakeIndex{}
	indexer := NewIndexer(nil, index, []domain.DocumentRoot{{ID: "documents", ServerPath: root}})

	// when
	err := indexer.RemoveFile(filePath)

	// then
	if err != nil {
		t.Fatalf("RemoveFile returned error: %v", err)
	}
	if index.deleted != domain.DocumentID("documents:nested/report.pdf") {
		t.Fatalf("deleted = %q, want logical document ID", index.deleted)
	}
}
