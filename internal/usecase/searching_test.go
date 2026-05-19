package usecase

import (
	"hwp-searcher/internal/domain"
	"testing"
)

func TestSearcherPrefersIgnoreSpacesMode(t *testing.T) {
	// Given
	index := &fakeIndex{}
	searcher := NewSearcher(index)

	// When
	result, err := searcher.Search("한 글", true, true)

	// Then
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("Total = %d, want 1", result.Total)
	}
	req := index.LastSearchRequest()
	if req.Mode != domain.SearchModeIgnoreSpaces {
		t.Fatalf("Search should prefer ignore-spaces mode when requested: mode = %v, want %v", req.Mode, domain.SearchModeIgnoreSpaces)
	}
	if req.Query != "한 글" {
		t.Fatalf("Search should pass through the user's query: query = %q, want %q", req.Query, "한 글")
	}
}

func TestSearcherRejectsQueriesShorterThanPersonNamePolicy(t *testing.T) {
	// Given
	index := &fakeIndex{}
	searcher := NewSearcher(index)

	// When
	_, err := searcher.Search("김", false, false)

	// Then
	if err == nil {
		t.Fatal("Search returned nil error, want short query error")
	}
	if req := index.LastSearchRequest(); req.Query != "" {
		t.Fatalf("Search should reject too-short queries before hitting the index: request = %+v", req)
	}
}
