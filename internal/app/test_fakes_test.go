package app

import "hwp-searcher/internal/domain"

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
