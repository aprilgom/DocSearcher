package main

import (
	"hwp-searcher/internal/search"
	"hwp-searcher/internal/server"
	"hwp-searcher/internal/watcher"
	"log"
)

func main() {
	// Initialize Search Index
	err := search.Init("hwp-index.bleve")
	if err != nil {
		log.Fatal("Failed to init index:", err)
	}

	// Start Watcher
	watcher.Start()

	// Start Server
	server.Start("8080")
}
