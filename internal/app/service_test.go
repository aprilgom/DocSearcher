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
	reset   bool
	count   uint64
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

func (f *fakeIndex) Count() (uint64, error) {
	return f.count, nil
}

func (f *fakeIndex) Reset() error {
	f.reset = true
	return nil
}

type fakeConfigStore struct {
	paths   []domain.WatchedPath
	added   domain.WatchedPath
	removed domain.WatchedPath
}

func (f *fakeConfigStore) WatchedPaths() []domain.WatchedPath {
	return f.paths
}

func (f *fakeConfigStore) AddPath(path domain.WatchedPath) error {
	f.added = path
	f.paths = append(f.paths, path)
	return nil
}

func (f *fakeConfigStore) RemovePath(path domain.WatchedPath) error {
	f.removed = path
	return nil
}

type fakeWatchRegistry struct {
	added   []domain.WatchedPath
	removed domain.WatchedPath
}

func (f *fakeWatchRegistry) AddPath(path domain.WatchedPath) error {
	f.added = append(f.added, path)
	return nil
}

func (f *fakeWatchRegistry) RemovePath(path domain.WatchedPath) error {
	f.removed = path
	return nil
}

type fakeIndexingStatus struct {
	indexing bool
}

func (f fakeIndexingStatus) IsIndexing() bool {
	return f.indexing
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

func TestServiceAddWatchedPathPersistsAndRegistersPath(t *testing.T) {
	store := &fakeConfigStore{}
	registry := &fakeWatchRegistry{}
	service := NewService(Dependencies{
		ConfigStore:   store,
		WatchRegistry: registry,
	})

	err := service.AddWatchedPath("/docs")

	if err != nil {
		t.Fatalf("AddWatchedPath returned error: %v", err)
	}
	if store.added != "/docs" {
		t.Fatalf("stored path = %q, want %q", store.added, "/docs")
	}
	if len(registry.added) != 1 || registry.added[0] != "/docs" {
		t.Fatalf("registered paths = %v, want [/docs]", registry.added)
	}
}

func TestServiceResetIndexReindexesWatchedPaths(t *testing.T) {
	index := &fakeIndex{}
	store := &fakeConfigStore{paths: []domain.WatchedPath{"/a", "/b"}}
	registry := &fakeWatchRegistry{}
	service := NewService(Dependencies{
		DocumentIndex: index,
		ConfigStore:   store,
		WatchRegistry: registry,
	})

	err := service.ResetIndex()

	if err != nil {
		t.Fatalf("ResetIndex returned error: %v", err)
	}
	if !index.reset {
		t.Fatal("Reset was not called")
	}
	if len(registry.added) != 2 || registry.added[0] != "/a" || registry.added[1] != "/b" {
		t.Fatalf("registered paths = %v, want [/a /b]", registry.added)
	}
}

func TestServiceStatsUsesPorts(t *testing.T) {
	index := &fakeIndex{count: 3}
	store := &fakeConfigStore{paths: []domain.WatchedPath{"/a", "/b"}}
	service := NewService(Dependencies{
		DocumentIndex:  index,
		ConfigStore:    store,
		IndexingStatus: fakeIndexingStatus{indexing: true},
	})

	stats, err := service.Stats()

	if err != nil {
		t.Fatalf("Stats returned error: %v", err)
	}
	if stats.DocumentCount != 3 || stats.WatchedPathCount != 2 || !stats.Indexing {
		t.Fatalf("stats = %+v, want count=3 watched=2 indexing=true", stats)
	}
}
