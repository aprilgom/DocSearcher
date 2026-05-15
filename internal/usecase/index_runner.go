package usecase

import (
	"hwp-searcher/internal/domain"
	"hwp-searcher/internal/infra/scanner"
	"hwp-searcher/internal/infra/worker"
	"log"
	"path/filepath"
	"strings"
	"sync/atomic"
)

type FileProcessor func(path string) error

type IndexRunner struct {
	processFile   FileProcessor
	workerPool    worker.Pool
	documentRoots []domain.DocumentRoot
	running       atomic.Bool
}

func NewIndexRunner(processFile FileProcessor, documentRoots ...[]domain.DocumentRoot) *IndexRunner {
	var roots []domain.DocumentRoot
	if len(documentRoots) > 0 {
		roots = append([]domain.DocumentRoot(nil), documentRoots[0]...)
	}
	return &IndexRunner{
		processFile:   processFile,
		workerPool:    worker.Pool{Size: 4},
		documentRoots: roots,
	}
}

func (r *IndexRunner) Start(root string) {
	if r.running.Load() {
		log.Println("Already indexing")
		return
	}
	r.running.Store(true)
	go func() {
		defer r.running.Store(false)
		log.Println("Starting index of", root)

		if err := r.Run(root); err != nil {
			log.Println("Walk error:", err)
		}
		log.Println("Indexing complete")
	}()
}

func (r *IndexRunner) Run(root string) error {
	jobs := make(chan string, 100)
	done := make(chan struct{})

	go func() {
		defer close(done)
		r.workerPool.Run(jobs, r.process)
	}()

	err := scanner.WalkWithOptions(root, scanner.WalkOptions{SkipDir: r.skipChildRoot(root)}, func(path string) error {
		jobs <- path
		return nil
	})

	close(jobs)
	<-done
	return err
}

func (r *IndexRunner) skipChildRoot(scanRoot string) func(path string) bool {
	if len(r.documentRoots) == 0 {
		return nil
	}
	cleanScanRoot := filepath.Clean(scanRoot)
	return func(path string) bool {
		cleanPath := filepath.Clean(path)
		if cleanPath == cleanScanRoot {
			return false
		}
		for _, root := range r.documentRoots {
			rootPath := filepath.Clean(root.ServerPath)
			if rootPath == cleanScanRoot || rootPath != cleanPath {
				continue
			}
			rel, err := filepath.Rel(cleanScanRoot, rootPath)
			if err != nil || filepath.IsAbs(rel) || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				continue
			}
			return true
		}
		return false
	}
}

func (r *IndexRunner) IsIndexing() bool {
	return r.running.Load()
}

func (r *IndexRunner) process(path string) {
	if err := r.processFile(path); err != nil {
		log.Printf("Failed to index %s: %v", path, err)
		return
	}
	log.Println("Indexed:", path)
}
