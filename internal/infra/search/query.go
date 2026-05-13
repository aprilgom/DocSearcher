package search

import (
	"hwp-searcher/internal/domain"

	"github.com/blevesearch/bleve/v2"
)

func buildSearchRequest(req domain.SearchRequest, schema domain.IndexSchema) *bleve.SearchRequest {
	var searchRequest *bleve.SearchRequest

	switch req.Mode {
	case domain.SearchModeIgnoreSpaces:
		query := bleve.NewMatchQuery(req.Query)
		query.FieldVal = schema.ContentNoSpaceField
		searchRequest = bleve.NewSearchRequest(query)
	case domain.SearchModeExact:
		query := bleve.NewMatchPhraseQuery(req.Query)
		query.FieldVal = schema.ContentField
		searchRequest = bleve.NewSearchRequest(query)
	default:
		query := bleve.NewQueryStringQuery(req.Query)
		searchRequest = bleve.NewSearchRequest(query)
	}

	searchRequest.Fields = []string{schema.PathField, schema.ContentField}
	searchRequest.Highlight = bleve.NewHighlight()
	return searchRequest
}
