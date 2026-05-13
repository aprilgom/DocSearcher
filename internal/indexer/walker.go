package indexer

import (
	"hwp-searcher/internal/domain"
	"hwp-searcher/internal/scanner"
	"log"
	"sync"
	"sync/atomic"
)

type FileIndexer interface {
	IndexFile(path string) error
	RemoveFile(path string) error
}

var (
	IndexedCount uint64
	IsIndexing   atomic.Bool
)

type Runner struct {
	fileIndexer FileIndexer
}

func NewRunner(fileIndexer FileIndexer) Runner {
	return Runner{fileIndexer: fileIndexer}
}

type Status struct{}

func (Status) IsIndexing() bool {
	return IsIndexing.Load()
}

func (Runner) IsIndexing() bool {
	return IsIndexing.Load()
}

func (r Runner) Start(root string) {
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
			go r.worker(jobs, &wg)
		}

		err := scanner.Walk(root, func(path string) error {
			jobs <- path
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

func (r Runner) worker(jobs <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for path := range jobs {
		r.IndexFile(path)
	}
}

func IsSupportedDocumentFile(path string) bool {
	return scanner.IsSupportedDocumentFile(path)
}

func NormalizeNoSpaceContent(content string) string {
	return domain.NormalizeNoSpaceContent(content)
}

// IndexFile indexes a single file
func (r Runner) IndexFile(path string) {
	err := r.fileIndexer.IndexFile(path)
	if err != nil {
		log.Printf("Failed to index %s: %v", path, err)
		return
	}
	atomic.AddUint64(&IndexedCount, 1)
	log.Println("Indexed:", path)
}

// RemoveFile removes a file from the index
func (r Runner) RemoveFile(path string) {
	err := r.fileIndexer.RemoveFile(path)
	if err != nil {
		log.Println("Failed to delete index:", path, err)
	} else {
		log.Println("Removed from index:", path)
	}
}
