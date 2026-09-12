package audit

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
)

type Job interface {
	Process() error
}

type Queue struct {
	jobs chan Job
	mu   sync.RWMutex
	done bool
	once sync.Once
}

func (r *RequestJob) Process() error {
	findings, err := engine.Evaluate(r, jobID)
	if err != nil {
		return err
	}

	if len(findings) == 0 {
		return nil
	}

	return store.SaveAuditResult(findings)
}

func (r *ResponseJob) Process() error {
	findings, err := engine.Evaluate(r, jobID)
	if err != nil {
		return err
	}

	if len(findings) == 0 {
		return nil
	}

	return store.SaveAuditResult(findings)
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

	for i := 0; i < count; i++ {
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

					if err := ProcessJob(ctx, job); err != nil {
						logger.Printf("audit worker %d failed to process job: %v", workerID, err)
					}
				}()
			}

			logger.Printf("audit worker %d queue closed", workerID)
		}(i)
	}

	return &wg
}

func ProcessJob(ctx context.Context, job Job) error {
	if job == nil {
		return fmt.Errorf("nil audit job")
	}

	return job.Process()
}

func (q *Queue) Close() {
	q.once.Do(func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		q.done = true
		close(q.jobs)
	})
}
