package usecase

import (
	"hwp-searcher/internal/infra/scanner"
	"hwp-searcher/internal/infra/worker"
	"log"
	"sync/atomic"
)

type FileProcessor func(path string) error

type IndexRunner struct {
	processFile FileProcessor
	workerPool  worker.Pool
	running     atomic.Bool
}

func NewIndexRunner(processFile FileProcessor) *IndexRunner {
	return &IndexRunner{
		processFile: processFile,
		workerPool:  worker.Pool{Size: 4},
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

	err := scanner.Walk(root, func(path string) error {
		jobs <- path
		return nil
	})

	close(jobs)
	<-done
	return err
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
