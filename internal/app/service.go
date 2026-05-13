package app

import "hwp-searcher/internal/domain"

type DocumentIndex interface {
	DocumentWriter
	DocumentSearcher
	DocumentCounter
	IndexResetter
}

type Dependencies struct {
	TextExtractor  TextExtractor
	DocumentIndex  DocumentIndex
	ConfigStore    ConfigStore
	WatchRegistry  WatchRegistry
	IndexingStatus IndexingStatus
}

type Service struct {
	indexer    Indexer
	searcher   Searcher
	watchPaths WatchPaths
	stats      Stats
	indexReset IndexResetter
}

func NewService(deps Dependencies) *Service {
	return &Service{
		indexer:    NewIndexer(deps.TextExtractor, deps.DocumentIndex),
		searcher:   NewSearcher(deps.DocumentIndex),
		watchPaths: NewWatchPaths(deps.ConfigStore, deps.WatchRegistry),
		stats:      NewStats(deps.DocumentIndex, deps.ConfigStore, deps.IndexingStatus),
		indexReset: deps.DocumentIndex,
	}
}

func (s *Service) IndexFile(path string) error {
	return s.indexer.IndexFile(path)
}

func (s *Service) RemoveFile(path string) error {
	return s.indexer.RemoveFile(path)
}

func (s *Service) Search(query string, exact bool, ignoreSpaces bool) (domain.SearchResult, error) {
	return s.searcher.Search(query, exact, ignoreSpaces)
}

func (s *Service) WatchedPaths() []domain.WatchedPath {
	return s.watchPaths.List()
}

func (s *Service) AddWatchedPath(path string) error {
	return s.watchPaths.Add(path)
}

func (s *Service) RemoveWatchedPath(path string) error {
	return s.watchPaths.Remove(path)
}

func (s *Service) ResetIndex() error {
	return s.watchPaths.ResetIndex(s.indexReset)
}

func (s *Service) Stats() (domain.Stats, error) {
	return s.stats.Current()
}
