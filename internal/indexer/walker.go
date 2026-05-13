package indexer

import (
	"hwp-searcher/internal/app"
	"hwp-searcher/internal/domain"
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
	service      = app.NewService(app.Dependencies{
		TextExtractor: parser.TextExtractor{},
		DocumentIndex: search.Engine{},
	})
)

func SetService(s *app.Service) {
	service = s
}

type Status struct{}

func (Status) IsIndexing() bool {
	return IsIndexing.Load()
}

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
			if !info.IsDir() && IsSupportedDocumentFile(path) {
				jobs <- path
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

func IsSupportedDocumentFile(path string) bool {
	name := filepath.Base(path)
	if strings.Contains(name, "~$") || strings.HasSuffix(strings.ToLower(name), ".tmp") {
		return false
	}

	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".hwp" || ext == ".hwpx" || ext == ".pdf"
}

func NormalizeNoSpaceContent(content string) string {
	return domain.NormalizeNoSpaceContent(content)
}

// IndexFile indexes a single file
func IndexFile(path string) {
	err := service.IndexFile(path)
	if err != nil {
		log.Printf("Failed to index %s: %v", path, err)
		return
	}
	atomic.AddUint64(&IndexedCount, 1)
	log.Println("Indexed:", path)
}

// RemoveFile removes a file from the index
func RemoveFile(path string) {
	err := service.RemoveFile(path)
	if err != nil {
		log.Println("Failed to delete index:", path, err)
	} else {
		log.Println("Removed from index:", path)
	}
}
