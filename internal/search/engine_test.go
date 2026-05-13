package search

import (
	"encoding/json"
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
