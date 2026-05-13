package server

import (
	"hwp-searcher/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeSearcher struct{}

func (fakeSearcher) Search(query string, exact bool, noSpace bool) (domain.SearchResult, error) {
	return domain.SearchResult{}, nil
}

type fakeWatchPaths struct{}

func (fakeWatchPaths) List() []domain.WatchedPath {
	return []domain.WatchedPath{"docs"}
}

func (fakeWatchPaths) Add(path string) error {
	return nil
}

func (fakeWatchPaths) Remove(path string) error {
	return nil
}

type fakeStats struct{}

func (fakeStats) Current() (domain.Stats, error) {
	return domain.Stats{DocumentCount: 2, WatchedPathCount: 1, Indexing: true}, nil
}

type fakeResetter struct{}

func (fakeResetter) ResetIndex() error {
	return nil
}

func TestNewMuxUsesInjectedHandlers(t *testing.T) {
	mux := NewMux(Handlers{
		Searcher:   fakeSearcher{},
		WatchPaths: fakeWatchPaths{},
		Stats:      fakeStats{},
		Resetter:   fakeResetter{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if got := rec.Body.String(); !strings.Contains(got, "2 docs | 1 watched | Indexing") {
		t.Fatalf("stats response = %q", got)
	}
}
