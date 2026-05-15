package search

import (
	"encoding/json"
	"hwp-searcher/internal/domain"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
	blevequery "github.com/blevesearch/bleve/v2/search/query"
)

const (
	testRootID       domain.RootID       = "documents"
	testRelativePath domain.RelativePath = "shared/2026/sample.hwp"
	testSearchText                       = "홍길동 보고서"
	testLegacyPath                       = "/legacy/report.hwp"
	testServerPath                       = "/srv/documents/shared/2026/sample.hwp"

	searchFixtureSingleTotal               uint64 = 1
	searchFixtureTotalWithCorruptHits      uint64 = 3
	searchFixtureTotalAfterCorruptOmission uint64 = 1
	searchFixturePageTotal                 uint64 = 7
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

func TestDocumentCodecFieldMapUsesSchemaFields(t *testing.T) {
	schema := domain.IndexSchema{
		ContentField:        "custom_body",
		ContentNoSpaceField: "custom_body_compact",
		PathField:           "custom_source_path",
		RootIDField:         "custom_root_id",
		RelativePathField:   "custom_relative_path",
		ServerPathField:     "custom_server_path",
	}
	codec := newDocumentCodec(schema)
	doc := domain.IndexedDocument{
		ID:             "reports/quarterly.hwp",
		Content:        "홍 길 동 보고서",
		ContentNoSpace: "홍길동보고서",
		Path:           "/ignored/by/codec.hwp",
		RootID:         "documents",
		RelativePath:   "reports/quarterly.hwp",
		ServerPath:     "/srv/docs/reports/quarterly.hwp",
	}

	got := codec.fieldMap(doc)
	want := map[string]string{
		schema.ContentField:        doc.Content,
		schema.ContentNoSpaceField: doc.ContentNoSpace,
		schema.PathField:           string(doc.ID),
		schema.RootIDField:         string(doc.RootID),
		schema.RelativePathField:   string(doc.RelativePath),
		schema.ServerPathField:     doc.ServerPath,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fieldMap() = %#v, want %#v", got, want)
	}
}

func TestBuildSearchRequestQueryModeUsesQueryStringAndResultOptions(t *testing.T) {
	schema := domain.IndexSchema{
		ContentField:        "body_text",
		ContentNoSpaceField: "body_text_compact",
		PathField:           "source_path",
		RootIDField:         "root_id",
		RelativePathField:   "relative_path",
		ServerPathField:     "server_path",
	}
	req := domain.SearchRequest{
		Query: "홍길동",
		Mode:  domain.SearchModeQuery,
	}

	got := buildSearchRequest(req, schema)

	query, ok := got.Query.(*blevequery.QueryStringQuery)
	if !ok {
		t.Fatalf("Query type = %T, want *query.QueryStringQuery", got.Query)
	}
	if query.Query != req.Query {
		t.Fatalf("Query.Query = %q, want %q", query.Query, req.Query)
	}
	if !reflect.DeepEqual(got.Fields, []string{schema.PathField, schema.RootIDField, schema.RelativePathField, schema.ServerPathField, schema.ContentField}) {
		t.Fatalf("Fields = %#v, want schema path/content fields", got.Fields)
	}
	if got.Highlight == nil {
		t.Fatalf("Highlight is nil, want configured highlight")
	}
}

func TestBuildSearchRequestExactModeTargetsContentField(t *testing.T) {
	schema := domain.IndexSchema{
		ContentField:        "body_text",
		ContentNoSpaceField: "body_text_compact",
		PathField:           "source_path",
		RootIDField:         "root_id",
		RelativePathField:   "relative_path",
	}
	req := domain.SearchRequest{
		Query: "홍길동 보고서",
		Mode:  domain.SearchModeExact,
	}

	got := buildSearchRequest(req, schema)

	query, ok := got.Query.(*blevequery.MatchPhraseQuery)
	if !ok {
		t.Fatalf("Query type = %T, want *query.MatchPhraseQuery", got.Query)
	}
	if query.MatchPhrase != req.Query {
		t.Fatalf("MatchPhrase = %q, want %q", query.MatchPhrase, req.Query)
	}
	if query.FieldVal != schema.ContentField {
		t.Fatalf("FieldVal = %q, want content field %q", query.FieldVal, schema.ContentField)
	}
}

func TestBuildSearchRequestIgnoreSpacesModeTargetsContentNoSpaceField(t *testing.T) {
	schema := domain.IndexSchema{
		ContentField:        "body_text",
		ContentNoSpaceField: "body_text_compact",
		PathField:           "source_path",
		RootIDField:         "root_id",
		RelativePathField:   "relative_path",
	}
	req := domain.SearchRequest{
		Query: "홍길동",
		Mode:  domain.SearchModeIgnoreSpaces,
	}

	got := buildSearchRequest(req, schema)

	query, ok := got.Query.(*blevequery.MatchQuery)
	if !ok {
		t.Fatalf("Query type = %T, want *query.MatchQuery", got.Query)
	}
	if query.Match != req.Query {
		t.Fatalf("Match = %q, want %q", query.Match, req.Query)
	}
	if query.FieldVal != schema.ContentNoSpaceField {
		t.Fatalf("FieldVal = %q, want no-space field %q", query.FieldVal, schema.ContentNoSpaceField)
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

	doc := logicalIndexedDocument("documents", "doc-1.hwp", "홍길동 문서")
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

	first := logicalIndexedDocument("documents", "before-reset.hwp", "초기 문서")
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

	second := logicalIndexedDocument("documents", "after-reset.hwp", "재색인 문서")
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
		logicalIndexedDocument("documents", "plain.hwp", "홍길동 보고서"),
		logicalIndexedDocument("documents", "spaced.hwp", "홍 길 동 보고서"),
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
			wantIDs: []domain.DocumentID{"documents:plain.hwp", "documents:spaced.hwp"},
		},
		{
			name: "exact phrase search",
			req: domain.SearchRequest{
				Query: "홍길동",
				Mode:  domain.SearchModeExact,
			},
			wantIDs: []domain.DocumentID{"documents:plain.hwp"},
		},
		{
			name: "ignore spaces search",
			req: domain.SearchRequest{
				Query: "홍길동",
				Mode:  domain.SearchModeIgnoreSpaces,
			},
			wantIDs: []domain.DocumentID{"documents:plain.hwp", "documents:spaced.hwp"},
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

func TestSearchHydratesHitsFromStoredLogicalFields(t *testing.T) {
	// given
	engine, err := NewEngine(filepath.Join(t.TempDir(), "logical.bleve"))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close()

	doc := searchFixtureDocument(t.TempDir())
	if err := engine.IndexDocument(doc); err != nil {
		t.Fatalf("IndexDocument: %v", err)
	}

	// when
	got, err := engine.Search(domain.SearchRequest{Query: "홍길동", Mode: domain.SearchModeQuery})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	// then
	if len(got.Hits) != 1 {
		t.Fatalf("len(Hits) = %d, want 1", len(got.Hits))
	}
	hit := got.Hits[0]
	if hit.ID != doc.ID {
		t.Fatalf("ID = %q, want %q", hit.ID, doc.ID)
	}
	if hit.RootID != doc.RootID {
		t.Fatalf("RootID = %q, want %q", hit.RootID, doc.RootID)
	}
	if hit.RelativePath != doc.RelativePath {
		t.Fatalf("RelativePath = %q, want %q", hit.RelativePath, doc.RelativePath)
	}
	if hit.Path != doc.ServerPath {
		t.Fatalf("Path = %q, want hydrated server path", hit.Path)
	}
}

func TestHitMapperOmitsHitsWithInvalidStoredLogicalFields(t *testing.T) {
	// given
	schema := domain.DefaultIndexSchema()
	mapper := newHitMapper(schema)

	result := &bleve.SearchResult{
		Total: searchFixtureTotalWithCorruptHits,
		Hits: search.DocumentMatchCollection{
			{
				ID: string(mustLogicalID(testRootID, "path.hwp")),
				Fields: map[string]interface{}{
					schema.RootIDField:       string(testRootID),
					schema.RelativePathField: "path.hwp",
					schema.ServerPathField:   "/srv/documents/path.hwp",
				},
			},
			{
				ID: string(mustLogicalID(testRootID, "missing.hwp")),
				Fields: map[string]interface{}{
					schema.RootIDField:       string(testRootID),
					schema.RelativePathField: "missing.hwp",
				},
			},
			{
				ID: string(mustLogicalID(testRootID, "invalid.hwp")),
				Fields: map[string]interface{}{
					schema.RootIDField:       "Documents",
					schema.RelativePathField: "../path.hwp",
				},
			},
		},
	}

	// when
	got := mapper.searchResult(result, domain.SearchRequest{Query: "path", Mode: domain.SearchModeQuery})

	// then
	if len(got.Hits) != 1 {
		t.Fatalf("len(Hits) = %d, want 1", len(got.Hits))
	}
	if got.Total != searchFixtureTotalAfterCorruptOmission {
		t.Fatalf("Total = %d, want total minus omitted corrupt hits", got.Total)
	}
	if got.Hits[0].ID != domain.DocumentID(mustLogicalID(testRootID, "path.hwp")) {
		t.Fatalf("hit ID = %q, want valid hit only", got.Hits[0].ID)
	}
}

func TestHitMapperKeepsLegacyPathHitsWithoutStoredLogicalFields(t *testing.T) {
	// given
	schema := domain.DefaultIndexSchema()
	mapper := newHitMapper(schema)

	result := &bleve.SearchResult{
		Total: searchFixtureSingleTotal,
		Hits: search.DocumentMatchCollection{
			{
				ID: testLegacyPath,
				Fragments: search.FieldFragmentMap{
					schema.ContentField: []string{"legacy fragment"},
				},
			},
		},
	}

	// when
	got := mapper.searchResult(result, domain.SearchRequest{Query: "legacy", Mode: domain.SearchModeQuery})

	// then
	if got.Total != searchFixtureSingleTotal || len(got.Hits) != int(searchFixtureSingleTotal) {
		t.Fatalf("result = %+v, want one legacy hit", got)
	}
	if got.Hits[0].ID != testLegacyPath || got.Hits[0].RelativePath != testLegacyPath || got.Hits[0].Path != testLegacyPath {
		t.Fatalf("hit = %+v, want legacy path identity", got.Hits[0])
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
				Fields: map[string]interface{}{
					schema.RootIDField:       "documents",
					schema.RelativePathField: "spaced.hwp",
					schema.ServerPathField:   testServerPath,
				},
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

func TestHitMapperMapsTotalIDsAndContentFragment(t *testing.T) {
	schema := domain.IndexSchema{
		ContentField:        "body",
		ContentNoSpaceField: "body_compact",
		PathField:           "source_path",
		RootIDField:         "root_id",
		RelativePathField:   "relative_path",
		ServerPathField:     "server_path",
	}
	mapper := newHitMapper(schema)

	result := &bleve.SearchResult{
		Total: searchFixturePageTotal,
		Hits: search.DocumentMatchCollection{
			{
				ID: "first.hwp",
				Fields: map[string]interface{}{
					schema.RootIDField:       string(testRootID),
					schema.RelativePathField: "first.hwp",
					schema.ServerPathField:   testServerPath,
				},
				Fragments: search.FieldFragmentMap{
					schema.ContentField: []string{"first fragment"},
				},
			},
			{
				ID: "second.pdf",
				Fields: map[string]interface{}{
					schema.RootIDField:       string(testRootID),
					schema.RelativePathField: "second.pdf",
					schema.ServerPathField:   testServerPath,
				},
				Fragments: search.FieldFragmentMap{
					schema.ContentField: []string{"second fragment"},
				},
			},
		},
	}

	got := mapper.searchResult(result, domain.SearchRequest{
		Query: "fragment",
		Mode:  domain.SearchModeExact,
	})

	if got.Total != searchFixturePageTotal {
		t.Fatalf("Total = %d, want Bleve total when no hits are omitted", got.Total)
	}
	if len(got.Hits) != 2 {
		t.Fatalf("len(Hits) = %d, want 2", len(got.Hits))
	}
	if got.Hits[0].ID != "first.hwp" || got.Hits[1].ID != "second.pdf" {
		t.Fatalf("hit IDs = %v, want [first.hwp second.pdf]", hitIDs(got.Hits))
	}
	if got.Hits[0].Fragment != "first fragment" {
		t.Fatalf("first Fragment = %q, want content fragment", got.Hits[0].Fragment)
	}
	if got.Hits[1].Fragment != "second fragment" {
		t.Fatalf("second Fragment = %q, want content fragment", got.Hits[1].Fragment)
	}
}

func TestHitMapperUsesEmptyFragmentWhenNoContentFragmentExists(t *testing.T) {
	schema := domain.DefaultIndexSchema()
	mapper := newHitMapper(schema)

	result := &bleve.SearchResult{
		Total: 1,
		Hits: search.DocumentMatchCollection{
			{
				ID: "empty-fragment.hwp",
				Fields: map[string]interface{}{
					schema.RootIDField:       "documents",
					schema.RelativePathField: "empty-fragment.hwp",
					schema.ServerPathField:   testServerPath,
				},
				Fragments: search.FieldFragmentMap{
					schema.PathField: []string{"/not/a/content/fragment.hwp"},
				},
			},
		},
	}

	got := mapper.searchResult(result, domain.SearchRequest{
		Query: "missing",
		Mode:  domain.SearchModeQuery,
	})

	if len(got.Hits) != 1 {
		t.Fatalf("len(Hits) = %d, want 1", len(got.Hits))
	}
	if got.Hits[0].Fragment != "" {
		t.Fatalf("Fragment = %q, want empty string", got.Hits[0].Fragment)
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

func logicalIndexedDocument(rootID domain.RootID, relativePath domain.RelativePath, content string) domain.IndexedDocument {
	id, err := domain.NewLogicalDocumentID(rootID, relativePath)
	if err != nil {
		panic(err)
	}
	serverPath := filepath.Join("/srv/docs", filepath.FromSlash(string(relativePath)))
	return domain.IndexedDocument{
		ID:             domain.DocumentID(id),
		RootID:         rootID,
		RelativePath:   relativePath,
		Content:        content,
		ContentNoSpace: domain.NormalizeNoSpaceContent(content),
		Path:           serverPath,
		ServerPath:     serverPath,
	}
}

func searchFixtureDocument(root string) domain.IndexedDocument {
	serverPath := filepath.Join(root, filepath.FromSlash(string(testRelativePath)))
	return domain.IndexedDocument{
		ID:             domain.DocumentID(mustLogicalID(testRootID, testRelativePath)),
		RootID:         testRootID,
		RelativePath:   testRelativePath,
		Content:        testSearchText,
		ContentNoSpace: domain.NormalizeNoSpaceContent(testSearchText),
		Path:           "server path is diagnostic only",
		ServerPath:     serverPath,
	}
}

func mustLogicalID(rootID domain.RootID, relativePath domain.RelativePath) domain.LogicalDocumentID {
	id, err := domain.NewLogicalDocumentID(rootID, relativePath)
	if err != nil {
		panic(err)
	}
	return id
}
