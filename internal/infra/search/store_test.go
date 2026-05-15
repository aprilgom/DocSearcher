package search

import (
	"hwp-searcher/internal/domain"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexStoreOpenAndDelegatesDocumentOperations(t *testing.T) {
	store := newIndexStore(filepath.Join(t.TempDir(), "store.bleve"))
	if err := store.open(); err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.close()

	schema := domain.DefaultIndexSchema()
	codec := newDocumentCodec(schema)
	doc := logicalIndexedDocument("documents", "alpha.hwp", "홍길동 보고서")

	if count, err := store.count(); err != nil || count != 0 {
		t.Fatalf("count before index = %d, %v; want 0, nil", count, err)
	}
	if err := store.indexDocument(string(doc.ID), codec.fieldMap(doc)); err != nil {
		t.Fatalf("indexDocument: %v", err)
	}
	if count, err := store.count(); err != nil || count != 1 {
		t.Fatalf("count after index = %d, %v; want 1, nil", count, err)
	}

	result, err := store.search(buildSearchRequest(domain.SearchRequest{
		Query: "홍길동",
		Mode:  domain.SearchModeQuery,
	}, schema))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("search Total = %d, want 1", result.Total)
	}

	if err := store.deleteDocument(string(doc.ID)); err != nil {
		t.Fatalf("deleteDocument: %v", err)
	}
	if count, err := store.count(); err != nil || count != 0 {
		t.Fatalf("count after delete = %d, %v; want 0, nil", count, err)
	}
	if err := store.close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestIndexStoreClosedOperationsReturnClosedErrorsAndCloseIsIdempotent(t *testing.T) {
	store := newIndexStore(filepath.Join(t.TempDir(), "closed.bleve"))
	if err := store.open(); err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := store.close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := store.close(); err != nil {
		t.Fatalf("second close: %v", err)
	}

	schema := domain.DefaultIndexSchema()
	codec := newDocumentCodec(schema)
	doc := logicalIndexedDocument("documents", "closed.hwp", "닫힌 인덱스 문서")

	if _, err := store.count(); !isIndexClosedError(err) {
		t.Fatalf("count error = %v, want index is closed", err)
	}
	if _, err := store.search(buildSearchRequest(domain.SearchRequest{
		Query: "닫힌",
		Mode:  domain.SearchModeQuery,
	}, schema)); !isIndexClosedError(err) {
		t.Fatalf("search error = %v, want index is closed", err)
	}
	if err := store.indexDocument(string(doc.ID), codec.fieldMap(doc)); !isIndexClosedError(err) {
		t.Fatalf("indexDocument error = %v, want index is closed", err)
	}
	if err := store.deleteDocument(string(doc.ID)); !isIndexClosedError(err) {
		t.Fatalf("deleteDocument error = %v, want index is closed", err)
	}
}

func isIndexClosedError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "index is closed")
}
