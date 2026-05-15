package search

import (
	"hwp-searcher/internal/domain"
	"log"

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
	omitted := uint64(0)
	for _, hit := range result.Hits {
		rootID, relativePath, path, ok := m.logicalFields(hit)
		if !ok {
			omitted++
			continue
		}
		hits = append(hits, domain.SearchHit{
			ID:           domain.DocumentID(hit.ID),
			RootID:       rootID,
			RelativePath: relativePath,
			Path:         path,
			Fragment:     m.fragment(hit, req),
		})
	}

	total := result.Total
	if omitted > total {
		total = 0
	} else {
		total -= omitted
	}
	return domain.SearchResult{
		Total: total,
		Hits:  hits,
	}
}

func (m hitMapper) logicalFields(hit *search.DocumentMatch) (domain.RootID, domain.RelativePath, string, bool) {
	rootID, ok := stringField(hit.Fields, m.schema.RootIDField)
	if !ok {
		return legacyLogicalFields(hit.ID)
	}
	relativePath, ok := stringField(hit.Fields, m.schema.RelativePathField)
	if !ok {
		return legacyLogicalFields(hit.ID)
	}

	root := domain.RootID(rootID)
	rel := domain.RelativePath(relativePath)
	if err := domain.ValidateRootID(root); err != nil {
		log.Printf("Skipping corrupt search hit %q: invalid root_id: %v", hit.ID, err)
		return "", "", "", false
	}
	if err := domain.ValidateRelativePath(rel); err != nil {
		log.Printf("Skipping corrupt search hit %q: invalid relative_path: %v", hit.ID, err)
		return "", "", "", false
	}
	serverPath, ok := stringField(hit.Fields, m.schema.ServerPathField)
	if !ok {
		log.Printf("Skipping corrupt search hit %q: missing server_path", hit.ID)
		return "", "", "", false
	}
	return root, rel, serverPath, true
}

func legacyLogicalFields(id string) (domain.RootID, domain.RelativePath, string, bool) {
	if rootID, relativePath, err := domain.ParseLogicalDocumentID(domain.LogicalDocumentID(id)); err == nil {
		log.Printf("Skipping corrupt search hit %q: missing stored logical fields", id)
		return rootID, relativePath, "", false
	}
	return "", domain.RelativePath(id), id, true
}

func stringField(fields map[string]interface{}, name string) (string, bool) {
	value, ok := fields[name]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	if !ok || text == "" {
		return "", false
	}
	return text, true
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
