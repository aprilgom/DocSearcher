package indexer

import (
	"hwp-searcher/internal/domain"
	"hwp-searcher/internal/scanner"
	"hwp-searcher/internal/worker"
	"log"
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
		done := make(chan struct{})
		go func() {
			defer close(done)
			worker.Pool{Size: 4}.Run(jobs, r.IndexFile)
		}()

		err := scanner.Walk(root, func(path string) error {
			jobs <- path
			return nil
		})
		if err != nil {
			log.Println("Walk error:", err)
		}

		close(jobs)
		<-done
		log.Println("Indexing complete")
	}()
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
