package app

import "hwp-searcher/internal/domain"

type DocumentSearcher interface {
	Search(req domain.SearchRequest) (domain.SearchResult, error)
}

type Searcher struct {
	documentIndex DocumentSearcher
}

func NewSearcher(documentIndex DocumentSearcher) Searcher {
	return Searcher{documentIndex: documentIndex}
}

func (s Searcher) Search(query string, exact bool, ignoreSpaces bool) (domain.SearchResult, error) {
	req := domain.SearchRequest{
		Query: query,
		Mode:  domain.SearchModeFromFlags(exact, ignoreSpaces),
	}
	if err := req.Validate(domain.PersonNameSearchPolicy()); err != nil {
		return domain.SearchResult{}, err
	}
	return s.documentIndex.Search(req)
}
