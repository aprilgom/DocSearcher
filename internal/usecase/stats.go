package usecase

import "hwp-searcher/internal/domain"

type DocumentCounter interface {
	Count() (uint64, error)
}

type IndexingStatus interface {
	IsIndexing() bool
}

type Stats struct {
	documentCounter DocumentCounter
	configStore     WatchPathReader
	indexingStatus  IndexingStatus
}

func NewStats(documentCounter DocumentCounter, configStore WatchPathReader, indexingStatus IndexingStatus) Stats {
	return Stats{
		documentCounter: documentCounter,
		configStore:     configStore,
		indexingStatus:  indexingStatus,
	}
}

func (s Stats) Current() (domain.Stats, error) {
	count, err := s.documentCounter.Count()
	if err != nil {
		return domain.Stats{}, err
	}

	return domain.Stats{
		DocumentCount:    count,
		WatchedPathCount: len(s.configStore.WatchedPaths()),
		Indexing:         s.indexingStatus.IsIndexing(),
	}, nil
}
