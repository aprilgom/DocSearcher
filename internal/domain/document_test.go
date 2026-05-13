package domain

import "testing"

func TestNewIndexedDocumentNormalizesSearchText(t *testing.T) {
	doc := NewDocument("report.hwpx", "한 글\nA\tB\r C")

	indexed := NewIndexedDocument(doc)

	if indexed.ID != "report.hwpx" {
		t.Fatalf("ID = %q, want %q", indexed.ID, "report.hwpx")
	}
	if indexed.ContentNoSpace != "한글ABC" {
		t.Fatalf("ContentNoSpace = %q, want %q", indexed.ContentNoSpace, "한글ABC")
	}
}

func TestSearchModeFromFlagsPrefersIgnoreSpaces(t *testing.T) {
	mode := SearchModeFromFlags(true, true)

	if mode != SearchModeIgnoreSpaces {
		t.Fatalf("SearchModeFromFlags(true, true) = %v, want %v", mode, SearchModeIgnoreSpaces)
	}
}

func TestPersonNameSearchPolicyDefinesIndexRules(t *testing.T) {
	policy := PersonNameSearchPolicy()

	if policy.MinQueryLength != 2 {
		t.Fatalf("MinQueryLength = %d, want %d", policy.MinQueryLength, 2)
	}
	if policy.PartialMatchMinGram != 2 {
		t.Fatalf("PartialMatchMinGram = %d, want %d", policy.PartialMatchMinGram, 2)
	}
	if policy.PartialMatchMaxGram != 5 {
		t.Fatalf("PartialMatchMaxGram = %d, want %d", policy.PartialMatchMaxGram, 5)
	}
	if !policy.IgnoreWhitespace {
		t.Fatal("IgnoreWhitespace = false, want true")
	}
}

func TestIndexSchemaDefinesSearchFields(t *testing.T) {
	schema := DefaultIndexSchema()

	if schema.ContentField != "content" {
		t.Fatalf("ContentField = %q, want %q", schema.ContentField, "content")
	}
	if schema.ContentNoSpaceField != "content_nospace" {
		t.Fatalf("ContentNoSpaceField = %q, want %q", schema.ContentNoSpaceField, "content_nospace")
	}
	if schema.PathField != "path" {
		t.Fatalf("PathField = %q, want %q", schema.PathField, "path")
	}
}

func TestSearchRequestValidateRejectsQueriesShorterThanPolicy(t *testing.T) {
	req := SearchRequest{Query: "김", Mode: SearchModeQuery}

	err := req.Validate(PersonNameSearchPolicy())

	if err == nil {
		t.Fatal("Validate returned nil, want error")
	}
}

func TestSearchRequestValidateAllowsPolicyLengthQuery(t *testing.T) {
	req := SearchRequest{Query: "김철", Mode: SearchModeQuery}

	if err := req.Validate(PersonNameSearchPolicy()); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}
