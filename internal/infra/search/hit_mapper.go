package search

import (
	"hwp-searcher/internal/domain"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
)

type hitMapper struct {
	schema domain.IndexSchema
}

func newHitMapper(schema domain.IndexSchema) hitMapper {
	return hitMapper{schema: schema}
}

func (m hitMapper) searchResult(result *bleve.SearchResult, req domain.SearchRequest) domain.SearchResult {
	hits := make([]domain.SearchHit, 0, len(result.Hits))
	for _, hit := range result.Hits {
		hits = append(hits, domain.SearchHit{
			ID:       domain.DocumentID(hit.ID),
			Fragment: m.fragment(hit, req),
		})
	}

	return domain.SearchResult{
		Total: result.Total,
		Hits:  hits,
	}
}

func (m hitMapper) fragment(hit *search.DocumentMatch, req domain.SearchRequest) string {
	fragment := ""
	if len(hit.Fragments[m.schema.ContentField]) > 0 {
		fragment = hit.Fragments[m.schema.ContentField][0]
	}
	if req.Mode == domain.SearchModeIgnoreSpaces && len(hit.Fragments[m.schema.ContentNoSpaceField]) > 0 {
		fragment = hit.Fragments[m.schema.ContentNoSpaceField][0]
	}
	return fragment
}
