package search

import (
	"fmt"
	"hwp-searcher/internal/domain"
	"os"
	"reflect"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/token/ngram"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
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
	e.indexPath = indexPath

	var err error
	if _, statErr := os.Stat(indexPath); os.IsNotExist(statErr) {
		fmt.Println("Index does not exist, creating new...")
		indexMapping := buildIndexMapping()
		e.index, err = bleve.New(indexPath, indexMapping)
	} else {
		fmt.Println("Index exists, opening...")
		e.index, err = bleve.Open(indexPath)
		// Auto-recovery: If open fails, try to recreate
		if err != nil {
			fmt.Printf("Failed to open index: %v. Attempting to recreate...\n", err)
			removeErr := os.RemoveAll(indexPath)
			if removeErr != nil {
				return fmt.Errorf("failed to remove corrupted index: %w", removeErr)
			}
			indexMapping := buildIndexMapping()
			e.index, err = bleve.New(indexPath, indexMapping)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to open/create index: %w", err)
	}

	if e.index == nil {
		return fmt.Errorf("bleve returned nil index with no error")
	}

	return nil
}

func buildIndexMapping() mapping.IndexMapping {
	policy := domain.PersonNameSearchPolicy()
	schema := domain.DefaultIndexSchema()
	indexMapping := bleve.NewIndexMapping()

	// 1. Define N-gram Token Filter
	err := indexMapping.AddCustomTokenFilter("ngram_filter", map[string]interface{}{
		"type": ngram.Name,
		"min":  float64(policy.PartialMatchMinGram),
		"max":  float64(policy.PartialMatchMaxGram),
	})
	if err != nil {
		panic(err)
	}

	// 2. Define Lowercase Token Filter (alias)
	// We explicitly define it to ensure it's available as "lowercase"
	err = indexMapping.AddCustomTokenFilter("lowercase", map[string]interface{}{
		"type": lowercase.Name,
	})
	if err != nil {
		panic(err)
	}

	// 3. Define Custom Analyzer (Unicode Tokenizer + N-gram Filter + Lowercase)
	err = indexMapping.AddCustomAnalyzer("ngram_analyzer", map[string]interface{}{
		"type":      custom.Name,
		"tokenizer": unicode.Name,
		"token_filters": []string{
			"ngram_filter",
			"lowercase",
		},
	})
	if err != nil {
		panic(err)
	}

	// 4. Define Document Mapping
	docMapping := bleve.NewDocumentMapping()

	// Field: content (Uses N-gram Analyzer)
	contentFieldMapping := bleve.NewTextFieldMapping()
	contentFieldMapping.Analyzer = "ngram_analyzer"
	docMapping.AddFieldMappingsAt(schema.ContentField, contentFieldMapping)

	// Field: content_nospace (Uses N-gram Analyzer - for person-name search without spaces)
	nospaceFieldMapping := bleve.NewTextFieldMapping()
	nospaceFieldMapping.Analyzer = "ngram_analyzer"
	docMapping.AddFieldMappingsAt(schema.ContentNoSpaceField, nospaceFieldMapping)

	// Field: path (Stored, not analyzed for full text search usually, but good to have)
	pathFieldMapping := bleve.NewTextFieldMapping()
	pathFieldMapping.Store = true
	docMapping.AddFieldMappingsAt(schema.PathField, pathFieldMapping)

	indexMapping.DefaultMapping = docMapping

	return indexMapping
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
	exact := req.Mode == domain.SearchModeExact
	ignoreSpaces := req.Mode == domain.SearchModeIgnoreSpaces
	schema := domain.DefaultIndexSchema()

	result, err := e.search(req.Query, exact, ignoreSpaces)
	if err != nil {
		return domain.SearchResult{}, err
	}

	hits := make([]domain.SearchHit, 0, len(result.Hits))
	for _, hit := range result.Hits {
		fragment := ""
		if len(hit.Fragments[schema.ContentField]) > 0 {
			fragment = hit.Fragments[schema.ContentField][0]
		}
		if ignoreSpaces && len(hit.Fragments[schema.ContentNoSpaceField]) > 0 {
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

func (e *Engine) search(queryStr string, exactMatch bool, ignoreSpaces bool) (*bleve.SearchResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.index == nil {
		return nil, fmt.Errorf("index is closed")
	}

	req := domain.SearchRequest{
		Query: queryStr,
		Mode:  domain.SearchModeFromFlags(exactMatch, ignoreSpaces),
	}
	return e.index.Search(buildSearchRequest(req, domain.DefaultIndexSchema()))
}

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
		e.index.Close()
		e.index = nil
	}
	err := os.RemoveAll(indexPath)
	if err != nil {
		return err
	}

	// Re-init (Init locks, but we are already locked, so we need to call internal init or unlock)
	// Since Init is exported and locks, we should extract the logic or just unlock before calling Init.
	// But Init checks file existence which we just deleted.

	// Let's just inline the Init logic here to avoid deadlock or recursive lock issues,
	// OR unlock and call Init. Unlocking is safer if Init does complex things, but here it's fine.
	// However, if we unlock, another thread might jump in.
	// Better to extract internal init logic.

	indexMapping := buildIndexMapping()
	newIndex, err := bleve.New(indexPath, indexMapping)
	if err != nil {
		return err
	}
	e.indexPath = indexPath
	e.index = newIndex
	return nil
}
