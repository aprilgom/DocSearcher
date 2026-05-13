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

type Engine struct{}

var (
	index bleve.Index
	mu    sync.RWMutex
)

// Init initializes the Bleve index with N-gram mapping
func Init(indexPath string) error {
	mu.Lock()
	defer mu.Unlock()

	fmt.Printf("Initializing index at %s\n", indexPath)

	var err error
	if _, statErr := os.Stat(indexPath); os.IsNotExist(statErr) {
		fmt.Println("Index does not exist, creating new...")
		indexMapping := buildIndexMapping()
		index, err = bleve.New(indexPath, indexMapping)
	} else {
		fmt.Println("Index exists, opening...")
		index, err = bleve.Open(indexPath)
		// Auto-recovery: If open fails, try to recreate
		if err != nil {
			fmt.Printf("Failed to open index: %v. Attempting to recreate...\n", err)
			removeErr := os.RemoveAll(indexPath)
			if removeErr != nil {
				return fmt.Errorf("failed to remove corrupted index: %w", removeErr)
			}
			indexMapping := buildIndexMapping()
			index, err = bleve.New(indexPath, indexMapping)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to open/create index: %w", err)
	}

	if index == nil {
		return fmt.Errorf("bleve returned nil index with no error")
	}

	return nil
}

func buildIndexMapping() mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()

	// 1. Define N-gram Token Filter
	err := indexMapping.AddCustomTokenFilter("ngram_filter", map[string]interface{}{
		"type": ngram.Name,
		"min":  1.0,
		"max":  10.0,
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
	docMapping.AddFieldMappingsAt("content", contentFieldMapping)

	// Field: content_nospace (Uses Standard Analyzer - for "Ignore Spaces")
	nospaceFieldMapping := bleve.NewTextFieldMapping()
	nospaceFieldMapping.Analyzer = "standard"
	docMapping.AddFieldMappingsAt("content_nospace", nospaceFieldMapping)

	// Field: path (Stored, not analyzed for full text search usually, but good to have)
	pathFieldMapping := bleve.NewTextFieldMapping()
	pathFieldMapping.Store = true
	docMapping.AddFieldMappingsAt("path", pathFieldMapping)

	indexMapping.DefaultMapping = docMapping

	return indexMapping
}

// IndexDocument adds a document to the index
func IndexDocument(id string, content string, contentNoSpace string) error {
	mu.RLock()
	defer mu.RUnlock()

	// Robust nil check
	if index == nil || (reflect.ValueOf(index).Kind() == reflect.Ptr && reflect.ValueOf(index).IsNil()) {
		return fmt.Errorf("index is closed or nil")
	}

	data := struct {
		Content        string `json:"content"`
		ContentNoSpace string `json:"content_nospace"`
		Path           string `json:"path"`
	}{
		Content:        content,
		ContentNoSpace: contentNoSpace,
		Path:           id,
	}
	return index.Index(id, data)
}

func (Engine) IndexDocument(doc domain.IndexedDocument) error {
	return IndexDocument(string(doc.ID), doc.Content, doc.ContentNoSpace)
}

func (Engine) Search(req domain.SearchRequest) (domain.SearchResult, error) {
	exact := req.Mode == domain.SearchModeExact
	ignoreSpaces := req.Mode == domain.SearchModeIgnoreSpaces

	result, err := Search(req.Query, exact, ignoreSpaces)
	if err != nil {
		return domain.SearchResult{}, err
	}

	hits := make([]domain.SearchHit, 0, len(result.Hits))
	for _, hit := range result.Hits {
		fragment := ""
		if len(hit.Fragments["content"]) > 0 {
			fragment = hit.Fragments["content"][0]
		}
		if ignoreSpaces && len(hit.Fragments["content_nospace"]) > 0 {
			fragment = hit.Fragments["content_nospace"][0]
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

// Search performs a query based on options
func Search(queryStr string, exactMatch bool, ignoreSpaces bool) (*bleve.SearchResult, error) {
	mu.RLock()
	defer mu.RUnlock()
	if index == nil {
		return nil, fmt.Errorf("index is closed")
	}

	var searchRequest *bleve.SearchRequest

	if ignoreSpaces {
		query := bleve.NewMatchQuery(queryStr)
		query.FieldVal = "content_nospace"
		searchRequest = bleve.NewSearchRequest(query)
	} else if exactMatch {
		query := bleve.NewMatchPhraseQuery(queryStr)
		query.FieldVal = "content"
		searchRequest = bleve.NewSearchRequest(query)
	} else {
		query := bleve.NewQueryStringQuery(queryStr)
		searchRequest = bleve.NewSearchRequest(query)
	}

	searchRequest.Fields = []string{"path", "content"}
	searchRequest.Highlight = bleve.NewHighlight()
	return index.Search(searchRequest)
}

// Count returns the number of indexed documents
func Count() (uint64, error) {
	mu.RLock()
	defer mu.RUnlock()
	if index == nil {
		return 0, fmt.Errorf("index is closed")
	}
	return index.DocCount()
}

// DeleteDocument removes a document from the index
func DeleteDocument(id string) error {
	mu.RLock()
	defer mu.RUnlock()
	if index == nil {
		return fmt.Errorf("index is closed")
	}
	return index.Delete(id)
}

func (Engine) DeleteDocument(id domain.DocumentID) error {
	return DeleteDocument(string(id))
}

func (Engine) Count() (uint64, error) {
	return Count()
}

func (Engine) Reset() error {
	return Reset("hwp-index.bleve")
}

// Reset closes, deletes, and re-initializes the index
func Reset(indexPath string) error {
	mu.Lock()
	defer mu.Unlock()

	if index != nil {
		index.Close()
		index = nil
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
	index = newIndex
	return nil
}
