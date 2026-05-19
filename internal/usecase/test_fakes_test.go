package usecase

import (
	"testing"

	"hwp-searcher/internal/domain"
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

func (f *fakeExtractor) LastRequestedPath() string {
	return f.path
}

type fakeIndex struct {
	indexed *domain.IndexedDocument
	deleted domain.DocumentID
	search  domain.SearchRequest
	reset   bool
	count   uint64

	indexErr  error
	deleteErr error
	searchErr error
	countErr  error
	resetErr  error
}

func (f *fakeIndex) IndexDocument(doc domain.IndexedDocument) error {
	f.indexed = &doc
	return f.indexErr
}

func (f *fakeIndex) DeleteDocument(id domain.DocumentID) error {
	f.deleted = id
	return f.deleteErr
}

func (f *fakeIndex) Search(req domain.SearchRequest) (domain.SearchResult, error) {
	f.search = req
	if f.searchErr != nil {
		return domain.SearchResult{}, f.searchErr
	}
	return domain.SearchResult{Total: 1}, nil
}

func (f *fakeIndex) Count() (uint64, error) {
	return f.count, f.countErr
}

func (f *fakeIndex) Reset() error {
	f.reset = true
	return f.resetErr
}

func (f *fakeIndex) IndexedDocument() (domain.IndexedDocument, bool) {
	if f.indexed == nil {
		return domain.IndexedDocument{}, false
	}
	return *f.indexed, true
}

func (f *fakeIndex) DeletedID() domain.DocumentID {
	return f.deleted
}

func (f *fakeIndex) LastSearchRequest() domain.SearchRequest {
	return f.search
}

func (f *fakeIndex) WasReset() bool {
	return f.reset
}

type fakeConfigStore struct {
	paths   []domain.WatchedPath
	added   domain.WatchedPath
	removed domain.WatchedPath
	loaded  bool

	loadErr   error
	addErr    error
	removeErr error
}

func (f *fakeConfigStore) Load() error {
	f.loaded = true
	return f.loadErr
}

func (f *fakeConfigStore) WatchedPaths() []domain.WatchedPath {
	return f.paths
}

func (f *fakeConfigStore) AddPath(path domain.WatchedPath) error {
	f.added = path
	f.paths = append(f.paths, path)
	return f.addErr
}

func (f *fakeConfigStore) RemovePath(path domain.WatchedPath) error {
	f.removed = path
	return f.removeErr
}

func (f *fakeConfigStore) StoredPaths() []domain.WatchedPath {
	return append([]domain.WatchedPath(nil), f.paths...)
}

func (f *fakeConfigStore) WasLoaded() bool {
	return f.loaded
}

func (f *fakeConfigStore) HasStoredPath(path domain.WatchedPath) bool {
	for _, stored := range f.paths {
		if stored == path {
			return true
		}
	}
	return false
}

type fakeWatchRegistry struct {
	added   []domain.WatchedPath
	removed domain.WatchedPath

	addErr    error
	removeErr error
}

func (f *fakeWatchRegistry) AddPath(path domain.WatchedPath) error {
	f.added = append(f.added, path)
	return f.addErr
}

func (f *fakeWatchRegistry) RemovePath(path domain.WatchedPath) error {
	f.removed = path
	return f.removeErr
}

func (f *fakeWatchRegistry) RegisteredPaths() []domain.WatchedPath {
	return append([]domain.WatchedPath(nil), f.added...)
}

type fakeIndexingStatus struct {
	indexing bool
}

func (f fakeIndexingStatus) IsIndexing() bool {
	return f.indexing
}

func assertWatchedPaths(t *testing.T, got []domain.WatchedPath, want []domain.WatchedPath, scenario string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: watched paths length = %d, want %d; got %v, want %v", scenario, len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: watched paths[%d] = %q, want %q; got %v, want %v", scenario, i, got[i], want[i], got, want)
		}
	}
}
