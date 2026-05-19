package usecase

import (
	"errors"
	"hwp-searcher/internal/domain"
	"testing"
)

func TestIndexerIndexesExtractedDocument(t *testing.T) {
	// Given
	extractor := &fakeExtractor{text: "한 글"}
	index := &fakeIndex{}
	indexer := NewIndexer(extractor, index)

	// When
	err := indexer.IndexFile("report.hwp")

	// Then
	if err != nil {
		t.Fatalf("IndexFile returned error: %v", err)
	}
	if extractor.LastRequestedPath() != "report.hwp" {
		t.Fatalf("IndexFile should extract the requested path: got %q, want %q", extractor.LastRequestedPath(), "report.hwp")
	}
	doc, ok := index.IndexedDocument()
	if !ok {
		t.Fatal("IndexFile should write the extracted document to the index")
	}
	if doc.ID != "report.hwp" {
		t.Fatalf("indexed document ID = %q, want source path %q", doc.ID, "report.hwp")
	}
	if doc.ContentNoSpace != "한글" {
		t.Fatalf("indexed document no-space content = %q, want %q", doc.ContentNoSpace, "한글")
	}
}

func TestIndexerPropagatesExtractTextError(t *testing.T) {
	// Given
	wantErr := errors.New("extract failed")
	extractor := &fakeExtractor{err: wantErr}
	index := &fakeIndex{}
	indexer := NewIndexer(extractor, index)

	// When
	err := indexer.IndexFile("broken.hwp")

	// Then
	if !errors.Is(err, wantErr) {
		t.Fatalf("IndexFile error = %v, want extract error %v", err, wantErr)
	}
	if _, ok := index.IndexedDocument(); ok {
		t.Fatal("IndexFile should not index a document when text extraction fails")
	}
}

func TestIndexerPropagatesIndexDocumentError(t *testing.T) {
	// Given
	wantErr := errors.New("index failed")
	extractor := &fakeExtractor{text: "한 글"}
	index := &fakeIndex{indexErr: wantErr}
	indexer := NewIndexer(extractor, index)

	// When
	err := indexer.IndexFile("report.hwp")

	// Then
	if !errors.Is(err, wantErr) {
		t.Fatalf("IndexFile error = %v, want index error %v", err, wantErr)
	}
	doc, ok := index.IndexedDocument()
	if !ok {
		t.Fatal("IndexFile should pass extracted document to index before returning index error")
	}
	if doc.ID != "report.hwp" {
		t.Fatalf("indexed document ID = %q, want source path %q", doc.ID, "report.hwp")
	}
}

func TestIndexerRemovesDocument(t *testing.T) {
	// Given
	index := &fakeIndex{}
	indexer := NewIndexer(nil, index)

	// When
	err := indexer.RemoveFile("report.pdf")

	// Then
	if err != nil {
		t.Fatalf("RemoveFile returned error: %v", err)
	}
	if index.DeletedID() != domain.DocumentID("report.pdf") {
		t.Fatalf("RemoveFile should delete by document path: got %q, want %q", index.DeletedID(), "report.pdf")
	}
}

func TestIndexerPropagatesDeleteDocumentError(t *testing.T) {
	// Given
	wantErr := errors.New("delete failed")
	index := &fakeIndex{deleteErr: wantErr}
	indexer := NewIndexer(nil, index)

	// When
	err := indexer.RemoveFile("report.pdf")

	// Then
	if !errors.Is(err, wantErr) {
		t.Fatalf("RemoveFile error = %v, want delete error %v", err, wantErr)
	}
	if index.DeletedID() != domain.DocumentID("report.pdf") {
		t.Fatalf("RemoveFile should delete by document path before returning delete error: got %q, want %q", index.DeletedID(), "report.pdf")
	}
}
