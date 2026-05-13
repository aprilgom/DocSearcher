package server

import (
	"errors"
	"hwp-searcher/internal/domain"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type fakeSearcher struct {
	result domain.SearchResult
	err    error
}

func (f fakeSearcher) Search(query string, exact bool, noSpace bool) (domain.SearchResult, error) {
	return f.result, f.err
}

type fakeWatchPaths struct {
	paths     []domain.WatchedPath
	addErr    error
	removeErr error
}

func (f fakeWatchPaths) List() []domain.WatchedPath {
	if f.paths != nil {
		return f.paths
	}
	return []domain.WatchedPath{"docs"}
}

func (f fakeWatchPaths) Add(path string) error {
	return f.addErr
}

func (f fakeWatchPaths) Remove(path string) error {
	return f.removeErr
}

type fakeStats struct {
	stats domain.Stats
	err   error
}

func (f fakeStats) Current() (domain.Stats, error) {
	if f.err != nil {
		return domain.Stats{}, f.err
	}
	if f.stats != (domain.Stats{}) {
		return f.stats, nil
	}
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

func TestSearchHandlerEscapesDynamicHTML(t *testing.T) {
	mux := NewMux(Handlers{
		Searcher: fakeSearcher{
			result: domain.SearchResult{
				Total: 1,
				Hits: []domain.SearchHit{{
					ID:       domain.DocumentID(`C:\docs\evil');alert(1);//<script>.pdf`),
					Fragment: `<mark onclick="alert(1)">match</mark>`,
				}},
			},
		},
		WatchPaths: fakeWatchPaths{},
		Stats:      fakeStats{},
		Resetter:   fakeResetter{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=test", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, `<script>`) {
		t.Fatalf("search response contains unescaped script tag: %q", body)
	}
	if strings.Contains(body, `<mark onclick="alert(1)">`) {
		t.Fatalf("search response contains unescaped fragment HTML: %q", body)
	}
	if strings.Contains(body, `triggerOpen('C:\\docs\\evil');alert(1);//`) {
		t.Fatalf("search response allows onclick string injection: %q", body)
	}
	if !strings.Contains(body, `&lt;mark onclick=&#34;alert(1)&#34;&gt;match&lt;/mark&gt;`) {
		t.Fatalf("search response does not include escaped fragment, got %q", body)
	}
}

func TestConfigHandlerEscapesWatchedPathHTML(t *testing.T) {
	mux := NewMux(Handlers{
		Searcher:   fakeSearcher{},
		WatchPaths: fakeWatchPaths{paths: []domain.WatchedPath{`docs"><script>alert(1)</script>`}},
		Stats:      fakeStats{},
		Resetter:   fakeResetter{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, `<script>`) {
		t.Fatalf("config response contains unescaped script tag: %q", body)
	}
	if !strings.Contains(body, `docs&#34;&gt;&lt;script&gt;alert(1)&lt;/script&gt;`) {
		t.Fatalf("config response does not include escaped path text, got %q", body)
	}
	if !strings.Contains(body, url.QueryEscape(`docs"><script>alert(1)</script>`)) {
		t.Fatalf("config response does not URL-escape delete path, got %q", body)
	}
}

func TestWatchHandlerReportsAddError(t *testing.T) {
	mux := NewMux(Handlers{
		Searcher:   fakeSearcher{},
		WatchPaths: fakeWatchPaths{addErr: errors.New("add failed <unsafe>")},
		Stats:      fakeStats{},
		Resetter:   fakeResetter{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/watch", strings.NewReader("path=docs"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "<unsafe>") {
		t.Fatalf("watch add error is not escaped: %q", body)
	}
	if !strings.Contains(body, "Add failed: add failed &lt;unsafe&gt;") {
		t.Fatalf("watch add error not reported, got %q", body)
	}
}

func TestWatchHandlerReportsRemoveError(t *testing.T) {
	mux := NewMux(Handlers{
		Searcher:   fakeSearcher{},
		WatchPaths: fakeWatchPaths{removeErr: errors.New("remove failed <unsafe>")},
		Stats:      fakeStats{},
		Resetter:   fakeResetter{},
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/watch?path=docs", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "<unsafe>") {
		t.Fatalf("watch remove error is not escaped: %q", body)
	}
	if !strings.Contains(body, "Remove failed: remove failed &lt;unsafe&gt;") {
		t.Fatalf("watch remove error not reported, got %q", body)
	}
}

func TestStatsHandlerReportsCurrentError(t *testing.T) {
	mux := NewMux(Handlers{
		Searcher:   fakeSearcher{},
		WatchPaths: fakeWatchPaths{},
		Stats:      fakeStats{err: errors.New("stats failed <unsafe>")},
		Resetter:   fakeResetter{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "<unsafe>") {
		t.Fatalf("stats error is not escaped: %q", body)
	}
	if !strings.Contains(body, "Stats failed: stats failed &lt;unsafe&gt;") {
		t.Fatalf("stats error not reported, got %q", body)
	}
}
