package dashboard

import (
	"context"
	"fmt"
	"observer/internal/query"
	"os"
)

// type Service struct{}
//
// func NewService() *Service {
// 	return &Service{}
// }

func GetLogs(ctx context.Context, cursor int64, limit int) (query.LogPage, error) {
	jobLogPath := os.Getenv("API_OBSERVER_JOB_LOG")

	if jobLogPath == "" {
		jobLogPath = "./logs/jobs.jsonl"
	}

	jobLogs, err := query.ReadJobLog(
		ctx,
		jobLogPath,
		query.LogCursor(cursor),
		limit,
	)
	if err != nil {
		return query.LogPage{}, fmt.Errorf(
			"failed to read job log %q: %w",
			jobLogPath,
			err,
		)
	}

	return jobLogs, nil
}
