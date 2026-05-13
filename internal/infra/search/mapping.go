package search

import (
	"fmt"
	"hwp-searcher/internal/domain"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/token/ngram"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
)

func buildIndexMapping() (mapping.IndexMapping, error) {
	policy := domain.PersonNameSearchPolicy()
	schema := domain.DefaultIndexSchema()
	indexMapping := bleve.NewIndexMapping()

	err := indexMapping.AddCustomTokenFilter("ngram_filter", map[string]interface{}{
		"type": ngram.Name,
		"min":  float64(policy.PartialMatchMinGram),
		"max":  float64(policy.PartialMatchMaxGram),
	})
	if err != nil {
		return nil, fmt.Errorf("add ngram token filter: %w", err)
	}

	err = indexMapping.AddCustomTokenFilter("lowercase", map[string]interface{}{
		"type": lowercase.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("add lowercase token filter: %w", err)
	}

	err = indexMapping.AddCustomAnalyzer("ngram_analyzer", map[string]interface{}{
		"type":      custom.Name,
		"tokenizer": unicode.Name,
		"token_filters": []string{
			"ngram_filter",
			"lowercase",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("add ngram analyzer: %w", err)
	}

	docMapping := bleve.NewDocumentMapping()

	contentFieldMapping := bleve.NewTextFieldMapping()
	contentFieldMapping.Analyzer = "ngram_analyzer"
	docMapping.AddFieldMappingsAt(schema.ContentField, contentFieldMapping)

	nospaceFieldMapping := bleve.NewTextFieldMapping()
	nospaceFieldMapping.Analyzer = "ngram_analyzer"
	docMapping.AddFieldMappingsAt(schema.ContentNoSpaceField, nospaceFieldMapping)

	pathFieldMapping := bleve.NewTextFieldMapping()
	pathFieldMapping.Store = true
	docMapping.AddFieldMappingsAt(schema.PathField, pathFieldMapping)

	indexMapping.DefaultMapping = docMapping

	return indexMapping, nil
}
