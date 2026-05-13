package main

import (
	"hwp-searcher/internal/app"
	"hwp-searcher/internal/config"
	"hwp-searcher/internal/indexer"
	"hwp-searcher/internal/parser"
	"hwp-searcher/internal/search"
	"hwp-searcher/internal/server"
	"hwp-searcher/internal/watcher"
	"log"
)

type fileHandler struct {
	indexer app.Indexer
}

func (h fileHandler) IndexFile(path string) {
	if err := h.indexer.IndexFile(path); err != nil {
		log.Printf("Failed to index %s: %v", path, err)
	}
}

func (h fileHandler) RemoveFile(path string) {
	if err := h.indexer.RemoveFile(path); err != nil {
		log.Printf("Failed to delete index: %s %v", path, err)
	}
}

func main() {
	// Initialize Search Index
	err := search.Init("hwp-index.bleve")
	if err != nil {
		log.Fatal("Failed to init index:", err)
	}

	fileIndexer := app.NewIndexer(parser.TextExtractor{}, search.Engine{})
	watchPaths := app.NewWatchPaths(config.Store{}, watcher.Registry{StartIndexing: indexer.Start})
	indexer.SetIndexer(fileIndexer)
	watcher.SetFileHandler(fileHandler{indexer: fileIndexer})
	server.SetHandlers(server.Handlers{
		Searcher:   app.NewSearcher(search.Engine{}),
		WatchPaths: watchPaths,
		Stats:      app.NewStats(search.Engine{}, config.Store{}, indexer.Status{}),
		Resetter:   search.Engine{},
	})

	// Start Watcher
	watcher.Start(watcher.Registry{StartIndexing: indexer.Start})

	// Start Server
	server.Start("8080")
}
