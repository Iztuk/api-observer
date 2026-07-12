package audit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Job interface {
	JobType() JobType
	Metadata() Metadata
	Process(c context.Context, e *RuleEngine) error
}

type Queue struct {
	jobs chan Job
	mu   sync.RWMutex
	done bool
	once sync.Once
}

type HTTPExchangeEvent struct {
	HostName string        `json:"host"`
	Request  *RequestCopy  `json:"request,omitempty"`
	Response *ResponseCopy `json:"response,omitempty"`
	Failure  *FailureCopy  `json:"failure,omitempty"`
}

type RequestCopy struct {
	Method string      `json:"method"`
	URL    string      `json:"url"`
	Header http.Header `json:"header"`
	Body   []byte      `json:"body"`
}

type ResponseCopy struct {
	Request    *RequestCopy `json:"request"`
	StatusCode int          `json:"status_code"`
	Headers    http.Header  `json:"headers"`
	Body       []byte       `json:"body"`
}

type FailureCopy struct {
	Request *RequestCopy `json:"request"`
	Error   string       `json:"error"`
}

func (q *Queue) ProcessHTTPEvent(ctx context.Context, event HTTPExchangeEvent, engine RuleEngine) error {
	if strings.TrimSpace(event.HostName) == "" {
		return fmt.Errorf("http exchange event missing host")
	}

	eventCount := 0

	if event.Request != nil {
		eventCount++
	}

	if event.Response != nil {
		eventCount++
	}

	if event.Failure != nil {
		eventCount++
	}

	if eventCount == 0 {
		return fmt.Errorf("http exchange event has no request, response, or failure payload")
	}

	if eventCount > 1 {
		return fmt.Errorf("http exchange event must contain only one payload type")
	}

	switch {
	case event.Request != nil:
		return q.processRequestEvent(ctx, event.HostName, event.Request)

	case event.Response != nil:
		return processResponseEvent(ctx, event.HostName, event.Response)

	case event.Failure != nil:
		return processFailureEvent(ctx, event.HostName, event.Failure)

	default:
		return fmt.Errorf("unsupported http exchange event")
	}
}

func (q *Queue) processRequestEvent(ctx context.Context, host string, reqCopy *RequestCopy) error {
	if reqCopy == nil {
		return fmt.Errorf("request event missing request")
	}

	req, err := http.NewRequest(
		reqCopy.Method,
		reqCopy.URL,
		bytes.NewReader(reqCopy.Body),
	)
	if err != nil {
		return err
	}

	req.Header = reqCopy.Header.Clone()
	req.ContentLength = int64(len(reqCopy.Body))

	job := NewRequestJob(req, host, time.Now().UTC())

	job.Body = reqCopy.Body

	if ok := q.TryEnqueue(job); !ok {
		return fmt.Errorf("failed to enqueue job: Request ID: %s", job.Meta.RequestID)
	}

	return nil
}

func processResponseEvent(ctx context.Context, host string, resCopy *ResponseCopy) error {
	if resCopy == nil {
		return fmt.Errorf("response event missing response")
	}

	req, err := http.NewRequest(
		resCopy.Request.Method,
		resCopy.Request.URL,
		bytes.NewReader(resCopy.Request.Body),
	)
	if err != nil {
		return err
	}

	req.Header = resCopy.Request.Header.Clone()
	req.ContentLength = int64(len(resCopy.Request.Body))

	resp := &http.Response{
		Request:       req,
		StatusCode:    resCopy.StatusCode,
		Status:        fmt.Sprintf("%d %s", resCopy.StatusCode, http.StatusText(resCopy.StatusCode)),
		Header:        resCopy.Headers.Clone(),
		Body:          io.NopCloser(bytes.NewReader(resCopy.Body)),
		ContentLength: int64(len(resCopy.Body)),
	}

	job := NewResponseJob(resp, host)

	job.Body = resCopy.Body

	return nil
}

func processFailureEvent(ctx context.Context, host string, failCopy *FailureCopy) error {

	return nil
}

func (r *RequestJob) JobType() JobType {
	return RequestJobType
}

func (r *RequestJob) Metadata() Metadata {
	return r.Meta
}

func (r *RequestJob) Process(ctx context.Context, engine *RuleEngine) error {
	jobID := uuid.NewString()

	findings, err := engine.Evaluate(r, jobID)
	if err != nil {
		return err
	}

	if len(findings) == 0 {
		return nil
	}

	fmt.Println("These are the findings:\n", findings)
	return DatabaseStore.SaveAuditResult(ctx, r, jobID, findings)
}

func (r *ResponseJob) JobType() JobType {
	return ResponseJobType
}

func (r *ResponseJob) Metadata() Metadata {
	return r.Meta
}

func (r *ResponseJob) Process(ctx context.Context, engine *RuleEngine) error {
	jobID := uuid.NewString()

	findings, err := engine.Evaluate(r, jobID)
	if err != nil {
		return err
	}

	if len(findings) == 0 {
		return nil
	}

	return DatabaseStore.SaveAuditResult(ctx, r, jobID, findings)
}

func (r *FailureJob) JobType() JobType {
	return FailureJobType
}

func (r *FailureJob) Metadata() Metadata {
	return r.Meta
}

func (r *FailureJob) Process(ctx context.Context, engine *RuleEngine) error {
	jobID := uuid.NewString()

	findings, err := engine.Evaluate(r, jobID)
	if err != nil {
		return err
	}

	if len(findings) == 0 {
		return nil
	}

	return DatabaseStore.SaveAuditResult(ctx, r, jobID, findings)
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

func (q *Queue) StartWorkers(ctx context.Context, count int, logger *log.Logger, engine *RuleEngine) *sync.WaitGroup {
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

					if err := ProcessJob(ctx, job, engine); err != nil {
						logger.Printf("audit worker %d failed to process job: %v", workerID, err)
					}
				}()
			}

			logger.Printf("audit worker %d queue closed", workerID)
		}(i)
	}

	return &wg
}

func ProcessJob(ctx context.Context, job Job, engine *RuleEngine) error {
	if job == nil {
		return fmt.Errorf("nil audit job")
	}

	return job.Process(ctx, engine)
}

func (q *Queue) Close() {
	q.once.Do(func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		q.done = true
		close(q.jobs)
	})
}
