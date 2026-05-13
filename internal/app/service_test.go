package app

import (
	"hwp-searcher/internal/domain"
	"testing"
)

type fakeExtractor struct {
	text string
	err  error
	path string
}

func (f *fakeExtractor) ExtractText(path string) (string, error) {
	f.path = path
	return f.text, f.err
}

type fakeIndex struct {
	indexed *domain.IndexedDocument
	deleted domain.DocumentID
	search  domain.SearchRequest
}

func (f *fakeIndex) IndexDocument(doc domain.IndexedDocument) error {
	f.indexed = &doc
	return nil
}

func (f *fakeIndex) DeleteDocument(id domain.DocumentID) error {
	f.deleted = id
	return nil
}

func (f *fakeIndex) Search(req domain.SearchRequest) (domain.SearchResult, error) {
	f.search = req
	return domain.SearchResult{Total: 1}, nil
}

func TestServiceIndexFileExtractsAndIndexesDocument(t *testing.T) {
	extractor := &fakeExtractor{text: "한 글"}
	index := &fakeIndex{}
	service := NewService(Dependencies{
		TextExtractor: extractor,
		DocumentIndex: index,
	})

	err := service.IndexFile("report.hwp")

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

func TestServiceRemoveFileDeletesDocument(t *testing.T) {
	index := &fakeIndex{}
	service := NewService(Dependencies{DocumentIndex: index})

	err := service.RemoveFile("report.pdf")

	if err != nil {
		t.Fatalf("RemoveFile returned error: %v", err)
	}
	if index.deleted != "report.pdf" {
		t.Fatalf("deleted = %q, want %q", index.deleted, "report.pdf")
	}
}

func TestServiceSearchPrefersIgnoreSpacesMode(t *testing.T) {
	index := &fakeIndex{}
	service := NewService(Dependencies{DocumentIndex: index})

	result, err := service.Search("한 글", true, true)

	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("Total = %d, want 1", result.Total)
	}
	if index.search.Mode != domain.SearchModeIgnoreSpaces {
		t.Fatalf("search mode = %v, want %v", index.search.Mode, domain.SearchModeIgnoreSpaces)
	}
	if index.search.Query != "한 글" {
		t.Fatalf("query = %q, want %q", index.search.Query, "한 글")
	}
}
