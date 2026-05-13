package search

import (
	"fmt"
	"hwp-searcher/internal/domain"
	"path/filepath"
	"strings"

	"github.com/blevesearch/bleve/v2"
)

type Engine struct {
	store *indexStore
}

func NewEngine(indexPath string) (*Engine, error) {
	normalizedPath, err := normalizeIndexPath(indexPath)
	if err != nil {
		return nil, err
	}

	store := newIndexStore(normalizedPath)
	if err := store.open(); err != nil {
		return nil, err
	}
	return &Engine{store: store}, nil
}

func normalizeIndexPath(indexPath string) (string, error) {
	indexPath = strings.TrimSpace(indexPath)
	if indexPath == "" {
		return "", fmt.Errorf("index path is empty")
	}

	absPath, err := filepath.Abs(indexPath)
	if err != nil {
		return "", fmt.Errorf("resolve index path: %w", err)
	}
	cleanPath := filepath.Clean(absPath)
	rootPath := filepath.Clean(filepath.VolumeName(cleanPath) + string(filepath.Separator))
	if cleanPath == rootPath {
		return "", fmt.Errorf("index path must not be a filesystem root: %s", cleanPath)
	}
	if filepath.Ext(cleanPath) != ".bleve" {
		return "", fmt.Errorf("index path must use .bleve extension: %s", cleanPath)
	}

	return cleanPath, nil
}

func (e *Engine) indexDocument(id string, content string, contentNoSpace string) error {
	schema := domain.DefaultIndexSchema()
	return e.store.indexDocument(id, map[string]string{
		schema.ContentField:        content,
		schema.ContentNoSpaceField: contentNoSpace,
		schema.PathField:           id,
	})
}

func (e *Engine) IndexDocument(doc domain.IndexedDocument) error {
	return e.indexDocument(string(doc.ID), doc.Content, doc.ContentNoSpace)
}

func (e *Engine) Search(req domain.SearchRequest) (domain.SearchResult, error) {
	schema := domain.DefaultIndexSchema()

	result, err := e.search(req)
	if err != nil {
		return domain.SearchResult{}, err
	}

	hits := make([]domain.SearchHit, 0, len(result.Hits))
	for _, hit := range result.Hits {
		fragment := ""
		if len(hit.Fragments[schema.ContentField]) > 0 {
			fragment = hit.Fragments[schema.ContentField][0]
		}
		if req.Mode == domain.SearchModeIgnoreSpaces && len(hit.Fragments[schema.ContentNoSpaceField]) > 0 {
			fragment = hit.Fragments[schema.ContentNoSpaceField][0]
		}

		hits = append(hits, domain.SearchHit{
			ID:       domain.DocumentID(hit.ID),
			Fragment: fragment,
		})
	}

	return domain.SearchResult{
		Total: result.Total,
		Hits:  hits,
	}, nil
}

func (e *Engine) search(req domain.SearchRequest) (*bleve.SearchResult, error) {
	return e.store.search(buildSearchRequest(req, domain.DefaultIndexSchema()))
}

func (e *Engine) Count() (uint64, error) {
	return e.store.count()
}

func (e *Engine) deleteDocument(id string) error {
	return e.store.deleteDocument(id)
}

func (e *Engine) DeleteDocument(id domain.DocumentID) error {
	return e.deleteDocument(string(id))
}

func (e *Engine) Reset() error {
	return e.store.reset()
}

func (e *Engine) Close() error {
	return e.store.close()
}
