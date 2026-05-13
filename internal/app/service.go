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
}

type Dependencies struct {
	TextExtractor TextExtractor
	DocumentIndex DocumentIndex
}

type Service struct {
	textExtractor TextExtractor
	documentIndex DocumentIndex
}

func NewService(deps Dependencies) *Service {
	return &Service{
		textExtractor: deps.TextExtractor,
		documentIndex: deps.DocumentIndex,
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
