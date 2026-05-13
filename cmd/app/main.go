package main

import (
	"hwp-searcher/internal/infra/config"
	"hwp-searcher/internal/infra/parser"
	"hwp-searcher/internal/infra/search"
	"hwp-searcher/internal/infra/watcher"
	"hwp-searcher/internal/server"
	"hwp-searcher/internal/usecase"
	"log"
)

type fileHandler struct {
	indexer usecase.Indexer
}

type indexResetHandler struct {
	watchPaths usecase.WatchPaths
	resetter   usecase.IndexResetter
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

	fileIndexer := usecase.NewIndexer(parser.TextExtractor{}, searchEngine)
	indexRunner := usecase.NewIndexRunner(fileIndexer.IndexFile)
	watchRegistry := watcher.Registry{StartIndexing: indexRunner.Start}
	configStore := config.NewStore(config.ConfigFile)
	watchPaths := usecase.NewWatchPaths(configStore, watchRegistry)
	watcher.SetFileHandler(fileHandler{indexer: fileIndexer})

	handlers := server.Handlers{
		Searcher:   usecase.NewSearcher(searchEngine),
		WatchPaths: watchPaths,
		Stats:      usecase.NewStats(searchEngine, configStore, indexRunner),
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
