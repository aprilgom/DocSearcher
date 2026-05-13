package worker

import "sync"

type Processor func(path string)

type Pool struct {
	Size int
}

func (p Pool) Run(jobs <-chan string, process Processor) {
	size := p.Size
	if size <= 0 {
		size = 1
	}

	var wg sync.WaitGroup
	for i := 0; i < size; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				process(path)
			}
		}()
	}
	wg.Wait()
}
