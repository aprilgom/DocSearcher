package indexer

import (
	"hwp-searcher/internal/parser"
	"hwp-searcher/internal/search"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	IndexedCount uint64
	IsIndexing   atomic.Bool
)

func Start(root string) {
	if IsIndexing.Load() {
		log.Println("Already indexing")
		return
	}
	IsIndexing.Store(true)
	go func() {
		defer IsIndexing.Store(false)
		log.Println("Starting index of", root)

		jobs := make(chan string, 100)
		var wg sync.WaitGroup

		// Start workers
		for i := 0; i < 4; i++ { // 4 workers
			wg.Add(1)
			go worker(jobs, &wg)
		}

		// Walk files
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				ext := strings.ToLower(filepath.Ext(path))
				if ext == ".hwp" || ext == ".pdf" {
					jobs <- path
				}
			}
			return nil
		})
		if err != nil {
			log.Println("Walk error:", err)
		}

		close(jobs)
		wg.Wait()
		log.Println("Indexing complete")
	}()
}

func worker(jobs <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for path := range jobs {
		IndexFile(path)
	}
}

// IndexFile indexes a single file
func IndexFile(path string) {
	content, err := parser.Parse(path)
	if err != nil {
		log.Printf("Failed to parse %s: %v", path, err)
		return
	}

	// Generate No-Space Content
	contentNoSpace := strings.ReplaceAll(content, " ", "")
	contentNoSpace = strings.ReplaceAll(contentNoSpace, "\n", "")
	contentNoSpace = strings.ReplaceAll(contentNoSpace, "\t", "")
	contentNoSpace = strings.ReplaceAll(contentNoSpace, "\r", "")

	err = search.IndexDocument(path, content, contentNoSpace)
	if err != nil {
		log.Printf("Failed to index %s: %v", path, err)
		return
	}
	atomic.AddUint64(&IndexedCount, 1)
	log.Println("Indexed:", path)
}

// RemoveFile removes a file from the index
func RemoveFile(path string) {
	err := search.DeleteDocument(path)
	if err != nil {
		log.Println("Failed to delete index:", path, err)
	} else {
		log.Println("Removed from index:", path)
	}
}
