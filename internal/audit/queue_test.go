package audit

import (
	"context"
	"io"
	"log"
	"net/http"
	"testing"
)

func TestQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := log.New(io.Discard, "", 0)

	q := NewQueue(10)

	jobs := []Job{
		RequestJob{
			Method:        http.MethodGet,
			URL:           "/users",
			Header:        http.Header{},
			Body:          "",
			ContentLength: 0,
			Metadata: Metadata{
				RequestID: "request-1",
				Source:    "test",
			},
		},
		ResponseJob{
			StatusCode:    http.StatusOK,
			Header:        http.Header{},
			Body:          `{"status":"ok"}`,
			ContentLength: 15,
			Metadata: Metadata{
				RequestID: "request-1",
				Source:    "test",
			},
		},
	}

	wg := q.StartWorkers(ctx, 1, logger, logger)

	for i, job := range jobs {
		if ok := q.TryEnqueue(job); !ok {
			t.Fatalf("failed to enqueue job %d", i)
		}
	}

	q.Close()
	wg.Wait()
}
