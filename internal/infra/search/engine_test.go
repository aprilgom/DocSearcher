package search

import (
	"encoding/json"
	"hwp-searcher/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
)

func TestBuildIndexMappingUsesDomainSearchPolicy(t *testing.T) {
	mapping, err := buildIndexMapping()
	if err != nil {
		t.Fatalf("buildIndexMapping: %v", err)
	}
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
	mapping, err := buildIndexMapping()
	if err != nil {
		t.Fatalf("buildIndexMapping: %v", err)
	}
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

func TestNewEngineRejectsUnsafeIndexPaths(t *testing.T) {
	tests := []struct {
		name      string
		indexPath string
	}{
		{name: "empty path", indexPath: ""},
		{name: "root path", indexPath: filepath.VolumeName(os.TempDir()) + string(filepath.Separator)},
		{name: "non bleve path", indexPath: filepath.Join(t.TempDir(), "documents")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngine(tt.indexPath)
			if err == nil {
				_ = engine.Close()
				t.Fatalf("NewEngine(%q) succeeded, want error", tt.indexPath)
			}
		})
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

func TestNewEngineDoesNotRemoveInvalidExistingIndex(t *testing.T) {
	indexPath := filepath.Join(t.TempDir(), "invalid.bleve")
	if err := os.Mkdir(indexPath, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	markerPath := filepath.Join(indexPath, "marker.txt")
	if err := os.WriteFile(markerPath, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	engine, err := NewEngine(indexPath)
	if err == nil {
		_ = engine.Close()
		t.Fatalf("NewEngine succeeded, want error")
	}
	if _, statErr := os.Stat(markerPath); statErr != nil {
		t.Fatalf("existing index content was removed or changed: %v", statErr)
	}
}

func TestResetClearsIndexAndKeepsEngineUsable(t *testing.T) {
	engine, err := NewEngine(filepath.Join(t.TempDir(), "reset.bleve"))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close()

	first := domain.NewIndexedDocument(domain.NewDocument("before-reset.hwp", "초기 문서"))
	if err := engine.IndexDocument(first); err != nil {
		t.Fatalf("IndexDocument before reset: %v", err)
	}
	if count, err := engine.Count(); err != nil || count != 1 {
		t.Fatalf("Count before reset = %d, %v; want 1, nil", count, err)
	}

	if err := engine.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if count, err := engine.Count(); err != nil || count != 0 {
		t.Fatalf("Count after reset = %d, %v; want 0, nil", count, err)
	}

	second := domain.NewIndexedDocument(domain.NewDocument("after-reset.hwp", "재색인 문서"))
	if err := engine.IndexDocument(second); err != nil {
		t.Fatalf("IndexDocument after reset: %v", err)
	}
	if count, err := engine.Count(); err != nil || count != 1 {
		t.Fatalf("Count after reindex = %d, %v; want 1, nil", count, err)
	}
}

func TestSearchSupportsQueryModes(t *testing.T) {
	engine, err := NewEngine(filepath.Join(t.TempDir(), "search.bleve"))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close()

	for _, doc := range []domain.IndexedDocument{
		domain.NewIndexedDocument(domain.NewDocument("plain.hwp", "홍길동 보고서")),
		domain.NewIndexedDocument(domain.NewDocument("spaced.hwp", "홍 길 동 보고서")),
	} {
		if err := engine.IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument %s: %v", doc.ID, err)
		}
	}

	tests := []struct {
		name    string
		req     domain.SearchRequest
		wantIDs []domain.DocumentID
	}{
		{
			name: "query string search",
			req: domain.SearchRequest{
				Query: "홍길동",
				Mode:  domain.SearchModeQuery,
			},
			wantIDs: []domain.DocumentID{"plain.hwp", "spaced.hwp"},
		},
		{
			name: "exact phrase search",
			req: domain.SearchRequest{
				Query: "홍길동",
				Mode:  domain.SearchModeExact,
			},
			wantIDs: []domain.DocumentID{"plain.hwp"},
		},
		{
			name: "ignore spaces search",
			req: domain.SearchRequest{
				Query: "홍길동",
				Mode:  domain.SearchModeIgnoreSpaces,
			},
			wantIDs: []domain.DocumentID{"plain.hwp", "spaced.hwp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Search(tt.req)
			if err != nil {
				t.Fatalf("Search: %v", err)
			}
			if !hasExactlyIDs(got.Hits, tt.wantIDs) {
				t.Fatalf("hits = %v, want IDs %v", hitIDs(got.Hits), tt.wantIDs)
			}
		})
	}
}

func TestHitMapperPrefersNoSpaceFragmentForIgnoreSpacesSearch(t *testing.T) {
	schema := domain.DefaultIndexSchema()
	mapper := newHitMapper(schema)

	result := &bleve.SearchResult{
		Total: 1,
		Hits: search.DocumentMatchCollection{
			{
				ID: "spaced.hwp",
				Fragments: search.FieldFragmentMap{
					schema.ContentField:        []string{"홍 길 동 보고서"},
					schema.ContentNoSpaceField: []string{"홍길동 보고서"},
				},
			},
		},
	}

	got := mapper.searchResult(result, domain.SearchRequest{
		Query: "홍길동",
		Mode:  domain.SearchModeIgnoreSpaces,
	})

	if len(got.Hits) != 1 {
		t.Fatalf("len(Hits) = %d, want 1", len(got.Hits))
	}
	if got.Hits[0].Fragment != "홍길동 보고서" {
		t.Fatalf("Fragment = %q, want no-space fragment", got.Hits[0].Fragment)
	}
}

func hasExactlyIDs(hits []domain.SearchHit, want []domain.DocumentID) bool {
	got := hitIDs(hits)
	if len(got) != len(want) {
		return false
	}
	remaining := make(map[domain.DocumentID]int, len(want))
	for _, id := range want {
		remaining[id]++
	}
	for _, id := range got {
		if remaining[id] == 0 {
			return false
		}
		remaining[id]--
	}
	return true
}

func hitIDs(hits []domain.SearchHit) []domain.DocumentID {
	ids := make([]domain.DocumentID, 0, len(hits))
	for _, hit := range hits {
		ids = append(ids, hit.ID)
	}
	return ids
}
