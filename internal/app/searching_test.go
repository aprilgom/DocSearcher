package app

import (
	"hwp-searcher/internal/domain"
	"testing"
)

func TestSearcherPrefersIgnoreSpacesMode(t *testing.T) {
	index := &fakeIndex{}
	searcher := NewSearcher(index)

	result, err := searcher.Search("한 글", true, true)

	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("Total = %d, want 1", result.Total)
	}
	if index.search.Mode != domain.SearchModeIgnoreSpaces {
		t.Fatalf("search mode = %v, want %v", index.search.Mode, domain.SearchModeIgnoreSpaces)
	}
	if index.search.Query != "한 글" {
		t.Fatalf("query = %q, want %q", index.search.Query, "한 글")
	}
}

func TestSearcherRejectsQueriesShorterThanPersonNamePolicy(t *testing.T) {
	index := &fakeIndex{}
	searcher := NewSearcher(index)

	_, err := searcher.Search("김", false, false)

	if err == nil {
		t.Fatal("Search returned nil error, want short query error")
	}
	if index.search.Query != "" {
		t.Fatalf("index search was called with query %q", index.search.Query)
	}
}
