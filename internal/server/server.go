package server

import (
	"fmt"
	"html/template"
	"hwp-searcher/internal/domain"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Searcher interface {
	Search(query string, exact bool, noSpace bool) (domain.SearchResult, error)
}

type WatchPaths interface {
	List() []domain.WatchedPath
	Add(path string) error
	Remove(path string) error
}

type Stats interface {
	Current() (domain.Stats, error)
}

type IndexResetter interface {
	ResetIndex() error
}

type Handlers struct {
	Searcher   Searcher
	WatchPaths WatchPaths
	Stats      Stats
	Resetter   IndexResetter
}

var responseTemplates = template.Must(template.New("server-responses").Parse(`
{{define "error"}}<div class='text-red-500'>{{.}}</div>{{end}}
{{define "searchTime"}}<div id='search-time' hx-swap-oob='true'>{{.Duration}} ({{.Total}} hits)</div>{{end}}
{{define "noResults"}}<div class='text-center text-gray-500 py-10'>No results found</div>{{end}}
{{define "searchHit"}}
			<div class="bg-white p-4 rounded-lg shadow-sm border border-gray-100 hover:shadow-md transition">
				<div class="text-xs text-gray-500 mb-1">{{.ID}}</div>
				<div class="text-sm text-gray-800">{{.Fragment}}</div>
				<button class="mt-2 text-xs text-indigo-600 cursor-pointer hover:underline bg-transparent border-none p-0"
					onclick="triggerOpen('{{.Path}}')">Open File</button>
			</div>
{{end}}
{{define "watchedPath"}}
			<li class="flex justify-between items-center bg-gray-50 p-2 rounded">
				<span class="text-sm text-gray-700 truncate">{{.Path}}</span>
				<button hx-delete="/api/watch?path={{.DeletePath}}" hx-target="#watched-list" class="text-red-500 hover:text-red-700 text-xs font-bold px-2">
					Remove
				</button>
			</li>
{{end}}
{{define "stats"}}<span>{{.DocumentCount}} docs | {{.WatchedPathCount}} watched | {{.Status}}</span>{{end}}
{{define "resetSuccess"}}<div class='text-green-600'>Index reset! Re-indexing started...</div>{{end}}
`))

func NewMux(handlers Handlers) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/api/search", searchHandler(handlers))
	mux.HandleFunc("/api/config", configHandler(handlers))
	mux.HandleFunc("/api/watch", watchHandler(handlers))
	mux.HandleFunc("/api/stats", statsHandler(handlers))
	mux.HandleFunc("/api/index/reset", resetHandler(handlers))
	return mux
}

func Start(port string, handlers Handlers) {
	log.Printf("Server starting on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, NewMux(handlers)))
}

func render(w http.ResponseWriter, name string, data any) {
	if err := responseTemplates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderError(w http.ResponseWriter, message string, err error) {
	render(w, "error", fmt.Sprintf("%s: %v", message, err))
}

func highlightFragment(fragment string) template.HTML {
	const (
		openMark  = "<mark>"
		closeMark = "</mark>"
	)

	var b strings.Builder
	openMarks := 0
	for len(fragment) > 0 {
		switch {
		case strings.HasPrefix(fragment, openMark):
			b.WriteString(openMark)
			openMarks++
			fragment = strings.TrimPrefix(fragment, openMark)
		case strings.HasPrefix(fragment, closeMark) && openMarks > 0:
			b.WriteString(closeMark)
			openMarks--
			fragment = strings.TrimPrefix(fragment, closeMark)
		default:
			nextOpen := strings.Index(fragment, openMark)
			nextClose := -1
			if openMarks > 0 {
				nextClose = strings.Index(fragment, closeMark)
			}
			next := -1
			if nextOpen >= 0 {
				next = nextOpen
			}
			if nextClose >= 0 && (next == -1 || nextClose < next) {
				next = nextClose
			}
			if next == -1 {
				b.WriteString(template.HTMLEscapeString(fragment))
				fragment = ""
				continue
			}
			b.WriteString(template.HTMLEscapeString(fragment[:next]))
			fragment = fragment[next:]
		}
	}
	return template.HTML(b.String())
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	tmpl.Execute(w, nil)
}

func searchHandler(handlers Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		exact := r.URL.Query().Get("exact") == "true"
		nospace := r.URL.Query().Get("nospace") == "true"

		start := time.Now()
		res, err := handlers.Searcher.Search(query, exact, nospace)
		duration := time.Since(start)

		if err != nil {
			renderError(w, "Error", err)
			return
		}

		// HTMX OOB swap for time
		render(w, "searchTime", struct {
			Duration time.Duration
			Total    uint64
		}{
			Duration: duration.Round(time.Millisecond),
			Total:    res.Total,
		})

		if res.Total == 0 {
			render(w, "noResults", nil)
			return
		}

		for _, hit := range res.Hits {
			render(w, "searchHit", struct {
				ID       domain.DocumentID
				Fragment template.HTML
				Path     string
			}{
				ID:       hit.ID,
				Fragment: highlightFragment(hit.Fragment),
				Path:     hit.Path,
			})
		}
	}
}

func configHandler(handlers Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Return list of watched paths as HTML list items
		for _, path := range handlers.WatchPaths.List() {
			render(w, "watchedPath", struct {
				Path       domain.WatchedPath
				DeletePath string
			}{
				Path:       path,
				DeletePath: url.QueryEscape(string(path)),
			})
		}
	}
}

func watchHandler(handlers Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.FormValue("path")
		if r.Method == "DELETE" {
			path = r.URL.Query().Get("path")
		}

		if path == "" {
			return
		}

		if r.Method == "POST" {
			if err := handlers.WatchPaths.Add(path); err != nil {
				renderError(w, "Add failed", err)
				return
			}
		} else if r.Method == "DELETE" {
			if err := handlers.WatchPaths.Remove(path); err != nil {
				renderError(w, "Remove failed", err)
				return
			}
		}

		// Re-render the list
		configHandler(handlers)(w, r)
	}
}

func statsHandler(handlers Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := handlers.Stats.Current()
		if err != nil {
			renderError(w, "Stats failed", err)
			return
		}
		status := "Idle"
		if stats.Indexing {
			status = "Indexing..."
		}
		render(w, "stats", struct {
			DocumentCount    uint64
			WatchedPathCount int
			Status           string
		}{
			DocumentCount:    stats.DocumentCount,
			WatchedPathCount: stats.WatchedPathCount,
			Status:           status,
		})
	}
}

func resetHandler(handlers Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			return
		}

		err := handlers.Resetter.ResetIndex()
		if err != nil {
			renderError(w, "Reset failed", err)
			return
		}

		render(w, "resetSuccess", nil)
	}
}
