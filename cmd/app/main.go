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

type indexResetHandler struct {
	watchPaths app.WatchPaths
	resetter   app.IndexResetter
}

func (h indexResetHandler) ResetIndex() error {
	return h.watchPaths.ResetIndex(h.resetter)
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
	searchEngine, err := search.NewEngine("hwp-index.bleve")
	if err != nil {
		log.Fatal("Failed to init index:", err)
	}
	defer searchEngine.Close()

	fileIndexer := app.NewIndexer(parser.TextExtractor{}, searchEngine)
	indexRunner := indexer.NewRunner(fileIndexer)
	watchRegistry := watcher.Registry{StartIndexing: indexRunner.Start}
	watchPaths := app.NewWatchPaths(config.Store{}, watchRegistry)
	watcher.SetFileHandler(fileHandler{indexer: fileIndexer})

	handlers := server.Handlers{
		Searcher:   app.NewSearcher(searchEngine),
		WatchPaths: watchPaths,
		Stats:      app.NewStats(searchEngine, config.Store{}, indexRunner),
		Resetter:   indexResetHandler{watchPaths: watchPaths, resetter: searchEngine},
	}

	// Start Watcher
	watcher.Start()
	if err := watchPaths.Start(); err != nil {
		log.Println("Failed to start watchers:", err)
	}

	// Start Server
	server.Start("8080", handlers)
}
