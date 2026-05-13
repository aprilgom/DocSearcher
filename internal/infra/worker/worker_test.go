package worker

import (
	"sort"
	"sync"
	"testing"
)

func TestPoolRunProcessesJobs(t *testing.T) {
	jobs := make(chan string)
	pool := Pool{Size: 4}

	var mu sync.Mutex
	var processed []string

	go func() {
		defer close(jobs)
		jobs <- "report.hwp"
		jobs <- "manual.pdf"
	}()

	pool.Run(jobs, func(path string) {
		mu.Lock()
		defer mu.Unlock()
		processed = append(processed, path)
	})

	sort.Strings(processed)
	want := []string{"manual.pdf", "report.hwp"}
	if len(processed) != len(want) {
		t.Fatalf("processed = %v, want %v", processed, want)
	}
	for i := range want {
		if processed[i] != want[i] {
			t.Fatalf("processed = %v, want %v", processed, want)
		}
	}
}

func TestPoolRunDefaultsToOneWorker(t *testing.T) {
	jobs := make(chan string, 1)
	jobs <- "report.hwp"
	close(jobs)

	var processed int
	Pool{}.Run(jobs, func(string) {
		processed++
	})

	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
}
