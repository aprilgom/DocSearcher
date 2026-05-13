package app

import (
	"hwp-searcher/internal/domain"
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
