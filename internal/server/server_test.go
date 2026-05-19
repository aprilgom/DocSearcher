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

type fakeResetter struct {
	err error
}

func (f fakeResetter) ResetIndex() error {
	return f.err
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

func TestSearchHandlerPreservesSafeHighlightMarkTags(t *testing.T) {
	mux := NewMux(Handlers{
		Searcher: fakeSearcher{
			result: domain.SearchResult{
				Total: 1,
				Hits: []domain.SearchHit{{
					ID:       domain.DocumentID("report.pdf"),
					Fragment: `before <mark>match</mark> after`,
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
	if !strings.Contains(body, `before <mark>match</mark> after`) {
		t.Fatalf("search response does not preserve safe mark highlight, got %q", body)
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

func TestHandlersReportEscapedErrors(t *testing.T) {
	tests := []struct {
		name     string
		handlers Handlers
		method   string
		target   string
		body     string
		want     string
	}{
		{
			name: "watch_add",
			handlers: Handlers{
				Searcher:   fakeSearcher{},
				WatchPaths: fakeWatchPaths{addErr: errors.New("add failed <unsafe>")},
				Stats:      fakeStats{},
				Resetter:   fakeResetter{},
			},
			method: http.MethodPost,
			target: "/api/watch",
			body:   "path=docs",
			want:   "Add failed: add failed &lt;unsafe&gt;",
		},
		{
			name: "watch_remove",
			handlers: Handlers{
				Searcher:   fakeSearcher{},
				WatchPaths: fakeWatchPaths{removeErr: errors.New("remove failed <unsafe>")},
				Stats:      fakeStats{},
				Resetter:   fakeResetter{},
			},
			method: http.MethodDelete,
			target: "/api/watch?path=docs",
			want:   "Remove failed: remove failed &lt;unsafe&gt;",
		},
		{
			name: "stats_current",
			handlers: Handlers{
				Searcher:   fakeSearcher{},
				WatchPaths: fakeWatchPaths{},
				Stats:      fakeStats{err: errors.New("stats failed <unsafe>")},
				Resetter:   fakeResetter{},
			},
			method: http.MethodGet,
			target: "/api/stats",
			want:   "Stats failed: stats failed &lt;unsafe&gt;",
		},
		{
			name: "search",
			handlers: Handlers{
				Searcher:   fakeSearcher{err: errors.New("search failed <unsafe>")},
				WatchPaths: fakeWatchPaths{},
				Stats:      fakeStats{},
				Resetter:   fakeResetter{},
			},
			method: http.MethodGet,
			target: "/api/search?q=test",
			want:   "Error: search failed &lt;unsafe&gt;",
		},
		{
			name: "reset_error",
			handlers: Handlers{
				Searcher:   fakeSearcher{},
				WatchPaths: fakeWatchPaths{},
				Stats:      fakeStats{},
				Resetter:   fakeResetter{err: errors.New("reset failed <unsafe>")},
			},
			method: http.MethodPost,
			target: "/api/index/reset",
			want:   "Reset failed: reset failed &lt;unsafe&gt;",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var body *strings.Reader
			if test.body != "" {
				body = strings.NewReader(test.body)
			} else {
				body = strings.NewReader("")
			}

			mux := NewMux(test.handlers)
			req := httptest.NewRequest(test.method, test.target, body)
			if test.method == http.MethodPost && test.target == "/api/watch" {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			got := rec.Body.String()
			if strings.Contains(got, "<unsafe>") {
				t.Fatalf("error is not escaped: %q", got)
			}
			if !strings.Contains(got, test.want) {
				t.Fatalf("error not reported, got %q, want substring %q", got, test.want)
			}
		})
	}
}

func TestHandlersRenderStatusMessages(t *testing.T) {
	tests := []struct {
		name      string
		handlers  Handlers
		method    string
		target    string
		want      []string
		wantEmpty bool
	}{
		{
			name: "search_no_results",
			handlers: Handlers{
				Searcher: fakeSearcher{result: domain.SearchResult{
					Total: 0,
					Hits:  nil,
				}},
				WatchPaths: fakeWatchPaths{},
				Stats:      fakeStats{},
				Resetter:   fakeResetter{},
			},
			method: http.MethodGet,
			target: "/api/search?q=missing",
			want:   []string{"0 hits", "No results found"},
		},
		{
			name: "stats_idle",
			handlers: Handlers{
				Searcher:   fakeSearcher{},
				WatchPaths: fakeWatchPaths{},
				Stats:      fakeStats{stats: domain.Stats{DocumentCount: 3, WatchedPathCount: 2, Indexing: false}},
				Resetter:   fakeResetter{},
			},
			method: http.MethodGet,
			target: "/api/stats",
			want:   []string{"3 docs | 2 watched | Idle"},
		},
		{
			name: "reset_get_empty",
			handlers: Handlers{
				Searcher:   fakeSearcher{},
				WatchPaths: fakeWatchPaths{},
				Stats:      fakeStats{},
				Resetter:   fakeResetter{err: errors.New("should not be called")},
			},
			method:    http.MethodGet,
			target:    "/api/index/reset",
			wantEmpty: true,
		},
		{
			name: "reset_success",
			handlers: Handlers{
				Searcher:   fakeSearcher{},
				WatchPaths: fakeWatchPaths{},
				Stats:      fakeStats{},
				Resetter:   fakeResetter{},
			},
			method: http.MethodPost,
			target: "/api/index/reset",
			want:   []string{"Index reset! Re-indexing started..."},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mux := NewMux(test.handlers)
			req := httptest.NewRequest(test.method, test.target, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			got := rec.Body.String()
			if test.wantEmpty {
				if got != "" {
					t.Fatalf("response body = %q, want empty", got)
				}
				return
			}
			for _, want := range test.want {
				if !strings.Contains(got, want) {
					t.Errorf("response body = %q, want substring %q", got, want)
				}
			}
		})
	}
}
