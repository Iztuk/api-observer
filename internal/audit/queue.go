package audit

import (
	"context"
	"log"
	"runtime/debug"
	"sync"
)

type Job interface {
	Process()
}

type Queue struct {
	jobs chan Job
	mu   sync.RWMutex
	done bool
	once sync.Once
}

func (j RequestJob) Process() {
	log.Printf("Processed: %v\n", j)
}

func (j ResponseJob) Process() {
	log.Printf("Processed: %v\n", j)
}

func NewQueue(size int) *Queue {
	return &Queue{
		jobs: make(chan Job, size),
	}
}

func (q *Queue) TryEnqueue(job Job) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.done {
		return false
	}

	if job == nil {
		return false
	}

	select {
	case q.jobs <- job:
		return true
	default:
		return false
	}
}

func (q *Queue) StartWorkers(ctx context.Context, count int, logger *log.Logger) *sync.WaitGroup {
	var wg sync.WaitGroup

	for i := range count {
		wg.Add(1)

		go func(workerID int) {
			defer wg.Done()

			for job := range q.jobs {
				func() {
					defer func() {
						if r := recover(); r != nil {
							logger.Printf(
								"audit worker %d panic: %v\n%s",
								workerID,
								r,
								debug.Stack(),
							)
						}
					}()

					job.Process()
				}()
			}

			logger.Printf("audit worker %d queue closed", workerID)
		}(i)
	}

	return &wg
}

func (q *Queue) Close() {
	q.once.Do(func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		q.done = true
		close(q.jobs)
	})
}
