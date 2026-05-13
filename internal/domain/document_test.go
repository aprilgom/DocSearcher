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
