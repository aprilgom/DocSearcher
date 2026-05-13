package app

import (
	"hwp-searcher/internal/domain"
)

type TextExtractor interface {
	ExtractText(path string) (string, error)
}

type DocumentIndex interface {
	IndexDocument(doc domain.IndexedDocument) error
	DeleteDocument(id domain.DocumentID) error
	Search(req domain.SearchRequest) (domain.SearchResult, error)
	Count() (uint64, error)
	Reset() error
}

type ConfigStore interface {
	WatchedPaths() []domain.WatchedPath
	AddPath(path domain.WatchedPath) error
	RemovePath(path domain.WatchedPath) error
}

type WatchRegistry interface {
	AddPath(path domain.WatchedPath) error
	RemovePath(path domain.WatchedPath) error
}

type IndexingStatus interface {
	IsIndexing() bool
}

type Dependencies struct {
	TextExtractor  TextExtractor
	DocumentIndex  DocumentIndex
	ConfigStore    ConfigStore
	WatchRegistry  WatchRegistry
	IndexingStatus IndexingStatus
}

type Service struct {
	textExtractor  TextExtractor
	documentIndex  DocumentIndex
	configStore    ConfigStore
	watchRegistry  WatchRegistry
	indexingStatus IndexingStatus
}

func NewService(deps Dependencies) *Service {
	return &Service{
		textExtractor:  deps.TextExtractor,
		documentIndex:  deps.DocumentIndex,
		configStore:    deps.ConfigStore,
		watchRegistry:  deps.WatchRegistry,
		indexingStatus: deps.IndexingStatus,
	}
}

func (s *Service) IndexFile(path string) error {
	content, err := s.textExtractor.ExtractText(path)
	if err != nil {
		return err
	}

	doc := domain.NewDocument(path, content)
	return s.documentIndex.IndexDocument(domain.NewIndexedDocument(doc))
}

func (s *Service) RemoveFile(path string) error {
	return s.documentIndex.DeleteDocument(domain.DocumentID(path))
}

func (s *Service) Search(query string, exact bool, ignoreSpaces bool) (domain.SearchResult, error) {
	req := domain.SearchRequest{
		Query: query,
		Mode:  domain.SearchModeFromFlags(exact, ignoreSpaces),
	}
	return s.documentIndex.Search(req)
}

func (s *Service) WatchedPaths() []domain.WatchedPath {
	return s.configStore.WatchedPaths()
}

func (s *Service) AddWatchedPath(path string) error {
	watchedPath := domain.WatchedPath(path)
	if err := s.configStore.AddPath(watchedPath); err != nil {
		return err
	}
	return s.watchRegistry.AddPath(watchedPath)
}

func (s *Service) RemoveWatchedPath(path string) error {
	watchedPath := domain.WatchedPath(path)
	if err := s.configStore.RemovePath(watchedPath); err != nil {
		return err
	}
	return s.watchRegistry.RemovePath(watchedPath)
}

func (s *Service) ResetIndex() error {
	if err := s.documentIndex.Reset(); err != nil {
		return err
	}
	for _, path := range s.configStore.WatchedPaths() {
		if err := s.watchRegistry.AddPath(path); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Stats() (domain.Stats, error) {
	count, err := s.documentIndex.Count()
	if err != nil {
		return domain.Stats{}, err
	}

	return domain.Stats{
		DocumentCount:    count,
		WatchedPathCount: len(s.configStore.WatchedPaths()),
		Indexing:         s.indexingStatus.IsIndexing(),
	}, nil
}
