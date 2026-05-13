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
	service *app.Service
}

func (h fileHandler) IndexFile(path string) {
	if err := h.service.IndexFile(path); err != nil {
		log.Printf("Failed to index %s: %v", path, err)
	}
}

func (h fileHandler) RemoveFile(path string) {
	if err := h.service.RemoveFile(path); err != nil {
		log.Printf("Failed to delete index: %s %v", path, err)
	}
}

func main() {
	// Initialize Search Index
	err := search.Init("hwp-index.bleve")
	if err != nil {
		log.Fatal("Failed to init index:", err)
	}

	service := app.NewService(app.Dependencies{
		TextExtractor:  parser.TextExtractor{},
		DocumentIndex:  search.Engine{},
		ConfigStore:    config.Store{},
		WatchRegistry:  watcher.Registry{StartIndexing: indexer.Start},
		IndexingStatus: indexer.Status{},
	})
	indexer.SetService(service)
	watcher.SetFileHandler(fileHandler{service: service})
	server.SetService(service)

	// Start Watcher
	watcher.Start(watcher.Registry{StartIndexing: indexer.Start})

	// Start Server
	server.Start("8080")
}
