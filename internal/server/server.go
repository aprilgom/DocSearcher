package server

import (
	"fmt"
	"html/template"
	"hwp-searcher/internal/app"
	"hwp-searcher/internal/config"
	"hwp-searcher/internal/indexer"
	"hwp-searcher/internal/parser"
	"hwp-searcher/internal/search"
	"hwp-searcher/internal/watcher"
	"log"
	"net/http"
	"strings"
	"time"
)

var service = app.NewService(app.Dependencies{
	TextExtractor:  parser.TextExtractor{},
	DocumentIndex:  search.Engine{},
	ConfigStore:    config.Store{},
	WatchRegistry:  watcher.Registry{},
	IndexingStatus: indexer.Status{},
})

func Start(port string) {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/search", searchHandler)
	http.HandleFunc("/api/config", configHandler)
	http.HandleFunc("/api/watch", watchHandler)
	http.HandleFunc("/api/stats", statsHandler)
	http.HandleFunc("/api/index/reset", resetHandler)

	log.Printf("Server starting on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	tmpl.Execute(w, nil)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	exact := r.URL.Query().Get("exact") == "true"
	nospace := r.URL.Query().Get("nospace") == "true"

	start := time.Now()
	res, err := service.Search(query, exact, nospace)
	duration := time.Since(start)

	if err != nil {
		fmt.Fprintf(w, "<div class='text-red-500'>Error: %v</div>", err)
		return
	}

	// HTMX OOB swap for time
	fmt.Fprintf(w, "<div id='search-time' hx-swap-oob='true'>%s (%d hits)</div>", duration.Round(time.Millisecond), res.Total)

	if res.Total == 0 {
		fmt.Fprint(w, "<div class='text-center text-gray-500 py-10'>No results found</div>")
		return
	}

	for _, hit := range res.Hits {
		// Escape backslashes for JavaScript string
		escapedPath := strings.ReplaceAll(string(hit.ID), "\\", "\\\\")

		fmt.Fprintf(w, `
			<div class="bg-white p-4 rounded-lg shadow-sm border border-gray-100 hover:shadow-md transition">
				<div class="text-xs text-gray-500 mb-1">%s</div>
				<div class="text-sm text-gray-800">%s</div>
				<button class="mt-2 text-xs text-indigo-600 cursor-pointer hover:underline bg-transparent border-none p-0" 
					onclick="triggerOpen('%s')">Open File</button>
			</div>
		`, hit.ID, hit.Fragment, escapedPath)
	}
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	// Return list of watched paths as HTML list items
	for _, path := range service.WatchedPaths() {
		fmt.Fprintf(w, `
			<li class="flex justify-between items-center bg-gray-50 p-2 rounded">
				<span class="text-sm text-gray-700 truncate">%s</span>
				<button hx-delete="/api/watch?path=%s" hx-target="#watched-list" class="text-red-500 hover:text-red-700 text-xs font-bold px-2">
					Remove
				</button>
			</li>
		`, path, path)
	}
}

func watchHandler(w http.ResponseWriter, r *http.Request) {
	path := r.FormValue("path")
	if r.Method == "DELETE" {
		path = r.URL.Query().Get("path")
	}

	if path == "" {
		return
	}

	if r.Method == "POST" {
		_ = service.AddWatchedPath(path)
	} else if r.Method == "DELETE" {
		_ = service.RemoveWatchedPath(path)
	}

	// Re-render the list
	configHandler(w, r)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	stats, _ := service.Stats()
	status := "Idle"
	if stats.Indexing {
		status = "Indexing..."
	}
	fmt.Fprintf(w, "<span>%d docs | %d watched | %s</span>", stats.DocumentCount, stats.WatchedPathCount, status)
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}

	err := service.ResetIndex()
	if err != nil {
		fmt.Fprintf(w, "<div class='text-red-500'>Reset failed: %v</div>", err)
		return
	}

	fmt.Fprint(w, "<div class='text-green-600'>Index reset! Re-indexing started...</div>")
}
