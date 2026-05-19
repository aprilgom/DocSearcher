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

func TestBuildIndexMappingContractUsesDomainSearchPolicy(t *testing.T) {
	// Given / When
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

	analysis := mapValue(t, decoded, "analysis")
	tokenFilters := mapValue(t, analysis, "token_filters")
	ngramFilter := mapValue(t, tokenFilters, "ngram_filter")

	// Then
	if ngramFilter["min"] != float64(2) {
		t.Fatalf("index mapping ngram min = %v, want 2 for Korean search policy", ngramFilter["min"])
	}
	if ngramFilter["max"] != float64(5) {
		t.Fatalf("index mapping ngram max = %v, want 5 for Korean search policy", ngramFilter["max"])
	}
}

func TestBuildIndexMappingContractUsesDomainIndexSchemaFields(t *testing.T) {
	// Given / When
	mapping, err := buildIndexMapping()
	if err != nil {
		t.Fatalf("buildIndexMapping: %v", err)
	}
	raw, err := json.Marshal(mapping)
	if err != nil {
		t.Fatalf("Marshal mapping: %v", err)
	}

	text := string(raw)

	// Then
	for _, field := range []string{"content", "content_nospace", "path"} {
		if !strings.Contains(text, `"`+field+`"`) {
			t.Fatalf("index mapping should include schema field %q: %s", field, text)
		}
	}
}

func TestDocumentCodecContractMapsIndexedDocumentToSchemaFields(t *testing.T) {
	// Given
	schema := domain.IndexSchema{
		ContentField:        "custom_body",
		ContentNoSpaceField: "custom_body_compact",
		PathField:           "custom_source_path",
	}
	codec := newDocumentCodec(schema)
	doc := domain.IndexedDocument{
		ID:             "reports/quarterly.hwp",
		Content:        "홍 길 동 보고서",
		ContentNoSpace: "홍길동보고서",
		Path:           "/ignored/by/codec.hwp",
	}

	// When
	got := codec.fieldMap(doc)

	// Then
	want := map[string]string{
		schema.ContentField:        doc.Content,
		schema.ContentNoSpaceField: doc.ContentNoSpace,
		schema.PathField:           string(doc.ID),
	}
	assertStringMap(t, got, want, "document codec field map")
}

func TestNewEngineRejectsUnsafeIndexPaths(t *testing.T) {
	// Given
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
			// When
			engine, err := NewEngine(tt.indexPath)

			// Then
			if err == nil {
				_ = engine.Close()
				t.Fatalf("NewEngine(%q) succeeded, want unsafe index path error", tt.indexPath)
			}
		})
	}
}

func TestNewEngineCreatesIndependentIndexes(t *testing.T) {
	// Given
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

	// When
	firstCount, err := first.Count()
	if err != nil {
		t.Fatalf("first Count: %v", err)
	}
	secondCount, err := second.Count()
	if err != nil {
		t.Fatalf("second Count: %v", err)
	}

	// Then
	if firstCount != 1 {
		t.Fatalf("first index count = %d, want 1 indexed document", firstCount)
	}
	if secondCount != 0 {
		t.Fatalf("second index count = %d, want 0 because indexes should be independent", secondCount)
	}
}

func TestNewEngineDoesNotRemoveInvalidExistingIndex(t *testing.T) {
	// Given
	indexPath := filepath.Join(t.TempDir(), "invalid.bleve")
	if err := os.Mkdir(indexPath, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	markerPath := filepath.Join(indexPath, "marker.txt")
	if err := os.WriteFile(markerPath, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// When
	engine, err := NewEngine(indexPath)

	// Then
	if err == nil {
		_ = engine.Close()
		t.Fatalf("NewEngine succeeded, want error")
	}
	if _, statErr := os.Stat(markerPath); statErr != nil {
		t.Fatalf("existing index content was removed or changed: %v", statErr)
	}
}

func TestResetClearsIndexAndKeepsEngineUsable(t *testing.T) {
	// Given
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

	// When
	if err := engine.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	// Then
	if count, err := engine.Count(); err != nil || count != 0 {
		t.Fatalf("Count after reset = %d, %v; want empty index", count, err)
	}

	second := domain.NewIndexedDocument(domain.NewDocument("after-reset.hwp", "재색인 문서"))
	if err := engine.IndexDocument(second); err != nil {
		t.Fatalf("IndexDocument after reset: %v", err)
	}
	if count, err := engine.Count(); err != nil || count != 1 {
		t.Fatalf("Count after reindex = %d, %v; want engine to remain usable", count, err)
	}
}

func TestSearchSupportsQueryModes(t *testing.T) {
	// Given
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
		name               string
		req                domain.SearchRequest
		wantIDs            []domain.DocumentID
		wantMarkedFragment bool
	}{
		{
			name: "query string search",
			req: domain.SearchRequest{
				Query: "홍길동",
				Mode:  domain.SearchModeQuery,
			},
			wantIDs:            []domain.DocumentID{"plain.hwp", "spaced.hwp"},
			wantMarkedFragment: true,
		},
		{
			name: "exact phrase search",
			req: domain.SearchRequest{
				Query: "홍길동",
				Mode:  domain.SearchModeExact,
			},
			wantIDs:            []domain.DocumentID{"plain.hwp"},
			wantMarkedFragment: true,
		},
		{
			name: "ignore spaces search",
			req: domain.SearchRequest{
				Query: "홍길동",
				Mode:  domain.SearchModeIgnoreSpaces,
			},
			wantIDs:            []domain.DocumentID{"plain.hwp", "spaced.hwp"},
			wantMarkedFragment: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got, err := engine.Search(tt.req)

			// Then
			if err != nil {
				t.Fatalf("Search: %v", err)
			}
			if !hasExactlyIDs(got.Hits, tt.wantIDs) {
				t.Fatalf("%s hits = %v, want IDs %v", tt.name, hitIDs(got.Hits), tt.wantIDs)
			}
			if got.Total != uint64(len(tt.wantIDs)) {
				t.Fatalf("%s total = %d, want %d", tt.name, got.Total, len(tt.wantIDs))
			}
			if tt.wantMarkedFragment && !hasMarkedFragment(got.Hits) {
				t.Fatalf("%s fragments = %v, want at least one highlighted result fragment", tt.name, hitFragments(got.Hits))
			}
		})
	}
}

func TestSearchRejectsInvalidQueries(t *testing.T) {
	// Given
	engine, err := NewEngine(filepath.Join(t.TempDir(), "validate.bleve"))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close()

	tests := []struct {
		name  string
		query string
	}{
		{name: "empty query", query: ""},
		{name: "short query", query: "김"},
		{name: "blank query", query: " \t\n "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			_, err := engine.Search(domain.SearchRequest{
				Query: tt.query,
				Mode:  domain.SearchModeQuery,
			})

			// Then
			if err == nil {
				t.Fatalf("Search(%q) returned nil error, want validation error", tt.query)
			}
			if !strings.Contains(err.Error(), "query must be at least 2 characters") {
				t.Fatalf("Search(%q) error = %v, want minimum length validation error", tt.query, err)
			}
		})
	}
}

func TestDeleteDocumentRemovesIndexedDocument(t *testing.T) {
	// Given
	engine, err := NewEngine(filepath.Join(t.TempDir(), "delete.bleve"))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close()

	keep := domain.NewIndexedDocument(domain.NewDocument("keep.hwp", "홍길동 유지 문서"))
	remove := domain.NewIndexedDocument(domain.NewDocument("remove.hwp", "홍길동 삭제 문서"))
	for _, doc := range []domain.IndexedDocument{keep, remove} {
		if err := engine.IndexDocument(doc); err != nil {
			t.Fatalf("IndexDocument(%s): %v", doc.ID, err)
		}
	}

	// When
	if err := engine.DeleteDocument(remove.ID); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}

	// Then
	count, err := engine.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Fatalf("Count after delete = %d, want 1", count)
	}
	got, err := engine.Search(domain.SearchRequest{
		Query: "홍길동",
		Mode:  domain.SearchModeQuery,
	})
	if err != nil {
		t.Fatalf("Search after delete: %v", err)
	}
	if !hasExactlyIDs(got.Hits, []domain.DocumentID{keep.ID}) {
		t.Fatalf("Search after delete hits = %v, want only %s", hitIDs(got.Hits), keep.ID)
	}
}

func TestHitMapperContractPrefersNoSpaceFragmentForIgnoreSpacesSearch(t *testing.T) {
	// Given
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

	// When
	got := mapper.searchResult(result, domain.SearchRequest{
		Query: "홍길동",
		Mode:  domain.SearchModeIgnoreSpaces,
	})

	// Then
	if len(got.Hits) != 1 {
		t.Fatalf("len(Hits) = %d, want 1", len(got.Hits))
	}
	if got.Hits[0].Fragment != "홍길동 보고서" {
		t.Fatalf("ignore-spaces fragment = %q, want no-space highlighted fragment", got.Hits[0].Fragment)
	}
}

func TestHitMapperContractMapsTotalIDsAndContentFragment(t *testing.T) {
	// Given
	schema := domain.IndexSchema{
		ContentField:        "body",
		ContentNoSpaceField: "body_compact",
		PathField:           "source_path",
	}
	mapper := newHitMapper(schema)

	result := &bleve.SearchResult{
		Total: 7,
		Hits: search.DocumentMatchCollection{
			{
				ID: "first.hwp",
				Fragments: search.FieldFragmentMap{
					schema.ContentField: []string{"first fragment"},
				},
			},
			{
				ID: "second.pdf",
				Fragments: search.FieldFragmentMap{
					schema.ContentField: []string{"second fragment"},
				},
			},
		},
	}

	// When
	got := mapper.searchResult(result, domain.SearchRequest{
		Query: "fragment",
		Mode:  domain.SearchModeExact,
	})

	// Then
	if got.Total != result.Total {
		t.Fatalf("mapped total = %d, want Bleve total %d", got.Total, result.Total)
	}
	if len(got.Hits) != 2 {
		t.Fatalf("len(Hits) = %d, want 2", len(got.Hits))
	}
	if got.Hits[0].ID != "first.hwp" || got.Hits[1].ID != "second.pdf" {
		t.Fatalf("mapped hit IDs = %v, want [first.hwp second.pdf]", hitIDs(got.Hits))
	}
	if got.Hits[0].Fragment != "first fragment" {
		t.Fatalf("first mapped fragment = %q, want content fragment", got.Hits[0].Fragment)
	}
	if got.Hits[1].Fragment != "second fragment" {
		t.Fatalf("second mapped fragment = %q, want content fragment", got.Hits[1].Fragment)
	}
}

func TestHitMapperContractUsesEmptyFragmentWhenNoContentFragmentExists(t *testing.T) {
	// Given
	schema := domain.DefaultIndexSchema()
	mapper := newHitMapper(schema)

	result := &bleve.SearchResult{
		Total: 1,
		Hits: search.DocumentMatchCollection{
			{
				ID: "empty-fragment.hwp",
				Fragments: search.FieldFragmentMap{
					schema.PathField: []string{"/not/a/content/fragment.hwp"},
				},
			},
		},
	}

	// When
	got := mapper.searchResult(result, domain.SearchRequest{
		Query: "missing",
		Mode:  domain.SearchModeQuery,
	})

	// Then
	if len(got.Hits) != 1 {
		t.Fatalf("len(Hits) = %d, want 1", len(got.Hits))
	}
	if got.Hits[0].Fragment != "" {
		t.Fatalf("missing content fragment = %q, want empty string", got.Hits[0].Fragment)
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

func hitFragments(hits []domain.SearchHit) []string {
	fragments := make([]string, 0, len(hits))
	for _, hit := range hits {
		fragments = append(fragments, hit.Fragment)
	}
	return fragments
}

func hasMarkedFragment(hits []domain.SearchHit) bool {
	for _, hit := range hits {
		if strings.Contains(hit.Fragment, "<mark>") && strings.Contains(hit.Fragment, "</mark>") {
			return true
		}
	}
	return false
}

func mapValue(t *testing.T, values map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	value, ok := values[key]
	if !ok {
		t.Fatalf("mapping is missing key %q in %#v", key, values)
	}
	mapped, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("mapping key %q has type %T, want map[string]interface{}", key, value)
	}
	return mapped
}

func assertStringMap(t *testing.T, got map[string]string, want map[string]string, scenario string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d; got %#v, want %#v", scenario, len(got), len(want), got, want)
	}
	for key, wantValue := range want {
		gotValue, ok := got[key]
		if !ok {
			t.Fatalf("%s missing key %q; got %#v, want %#v", scenario, key, got, want)
		}
		if gotValue != wantValue {
			t.Fatalf("%s[%q] = %q, want %q; got %#v, want %#v", scenario, key, gotValue, wantValue, got, want)
		}
	}
}
