package search

import (
	"encoding/json"
	"hwp-searcher/internal/domain"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildIndexMappingUsesDomainSearchPolicy(t *testing.T) {
	mapping := buildIndexMapping()
	raw, err := json.Marshal(mapping)
	if err != nil {
		t.Fatalf("Marshal mapping: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal mapping: %v", err)
	}

	tokenFilters := decoded["analysis"].(map[string]interface{})["token_filters"].(map[string]interface{})
	ngramFilter := tokenFilters["ngram_filter"].(map[string]interface{})
	if ngramFilter["min"] != float64(2) {
		t.Fatalf("ngram min = %v, want 2", ngramFilter["min"])
	}
	if ngramFilter["max"] != float64(5) {
		t.Fatalf("ngram max = %v, want 5", ngramFilter["max"])
	}
}

func TestBuildIndexMappingUsesDomainIndexSchemaFields(t *testing.T) {
	mapping := buildIndexMapping()
	raw, err := json.Marshal(mapping)
	if err != nil {
		t.Fatalf("Marshal mapping: %v", err)
	}

	text := string(raw)
	for _, field := range []string{"content", "content_nospace", "path"} {
		if !strings.Contains(text, `"`+field+`"`) {
			t.Fatalf("mapping does not include field %q: %s", field, text)
		}
	}
}

func TestNewEngineCreatesIndependentIndexes(t *testing.T) {
	first, err := NewEngine(filepath.Join(t.TempDir(), "first.bleve"))
	if err != nil {
		t.Fatalf("NewEngine first: %v", err)
	}
	defer first.Close()

	second, err := NewEngine(filepath.Join(t.TempDir(), "second.bleve"))
	if err != nil {
		t.Fatalf("NewEngine second: %v", err)
	}
	defer second.Close()

	doc := domain.NewIndexedDocument(domain.NewDocument("doc-1.hwp", "홍길동 문서"))
	if err := first.IndexDocument(doc); err != nil {
		t.Fatalf("IndexDocument: %v", err)
	}

	firstCount, err := first.Count()
	if err != nil {
		t.Fatalf("first Count: %v", err)
	}
	secondCount, err := second.Count()
	if err != nil {
		t.Fatalf("second Count: %v", err)
	}

	if firstCount != 1 {
		t.Fatalf("first Count = %d, want 1", firstCount)
	}
	if secondCount != 0 {
		t.Fatalf("second Count = %d, want 0", secondCount)
	}
}
