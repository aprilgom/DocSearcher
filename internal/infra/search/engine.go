package search

import (
	"fmt"
	"hwp-searcher/internal/domain"
	"os"
	"reflect"
	"sync"

	"github.com/blevesearch/bleve/v2"
)

type Engine struct {
	indexPath string
	index     bleve.Index
	mu        sync.RWMutex
}

func NewEngine(indexPath string) (*Engine, error) {
	engine := &Engine{}
	if err := engine.Init(indexPath); err != nil {
		return nil, err
	}
	return engine, nil
}

// Init initializes the Bleve index with N-gram mapping
func (e *Engine) Init(indexPath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	fmt.Printf("Initializing index at %s\n", indexPath)
	index, err := openOrCreateIndex(indexPath)
	if err != nil {
		return fmt.Errorf("failed to open/create index: %w", err)
	}
	if index == nil {
		return fmt.Errorf("bleve returned nil index with no error")
	}

	e.indexPath = indexPath
	e.index = index
	return nil
}

func openOrCreateIndex(indexPath string) (bleve.Index, error) {
	if _, statErr := os.Stat(indexPath); os.IsNotExist(statErr) {
		fmt.Println("Index does not exist, creating new...")
		return createIndex(indexPath)
	}

	fmt.Println("Index exists, opening...")
	index, err := bleve.Open(indexPath)
	if err == nil {
		return index, nil
	}

	fmt.Printf("Failed to open index: %v. Attempting to recreate...\n", err)
	return recreateIndex(indexPath)
}

func recreateIndex(indexPath string) (bleve.Index, error) {
	if err := os.RemoveAll(indexPath); err != nil {
		return nil, fmt.Errorf("failed to remove corrupted index: %w", err)
	}
	return createIndex(indexPath)
}

func createIndex(indexPath string) (bleve.Index, error) {
	indexMapping, err := buildIndexMapping()
	if err != nil {
		return nil, err
	}
	return bleve.New(indexPath, indexMapping)
}

func (e *Engine) indexDocument(id string, content string, contentNoSpace string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Robust nil check
	if e.index == nil || (reflect.ValueOf(e.index).Kind() == reflect.Ptr && reflect.ValueOf(e.index).IsNil()) {
		return fmt.Errorf("index is closed or nil")
	}

	schema := domain.DefaultIndexSchema()
	return e.index.Index(id, map[string]string{
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
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.index == nil {
		return nil, fmt.Errorf("index is closed")
	}

	return e.index.Search(buildSearchRequest(req, domain.DefaultIndexSchema()))
}

func (e *Engine) Count() (uint64, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.index == nil {
		return 0, fmt.Errorf("index is closed")
	}
	return e.index.DocCount()
}

func (e *Engine) deleteDocument(id string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.index == nil {
		return fmt.Errorf("index is closed")
	}
	return e.index.Delete(id)
}

func (e *Engine) DeleteDocument(id domain.DocumentID) error {
	return e.deleteDocument(string(id))
}

func (e *Engine) Reset() error {
	return e.ResetIndex(e.indexPath)
}

func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.index == nil {
		return nil
	}
	err := e.index.Close()
	e.index = nil
	return err
}

// ResetIndex closes, deletes, and re-initializes the index
func (e *Engine) ResetIndex(indexPath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.index != nil {
		if err := e.index.Close(); err != nil {
			return err
		}
		e.index = nil
	}

	newIndex, err := recreateIndex(indexPath)
	if err != nil {
		return err
	}
	e.indexPath = indexPath
	e.index = newIndex
	return nil
}
