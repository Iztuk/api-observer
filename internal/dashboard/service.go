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

func GetLogs(ctx context.Context, queryString string, cursor int64, limit int) (query.LogPage, error) {
	jobLogPath := os.Getenv("API_OBSERVER_JOB_LOG")

	if jobLogPath == "" {
		jobLogPath = "./logs/jobs.jsonl"
	}

	findingsLogPath := os.Getenv("API_OBSERVER_FINDINGS_LOG")

	if findingsLogPath == "" {
		findingsLogPath = "./logs/findings.jsonl"
	}

	expr, err := query.ParseQuery(queryString)
	if err != nil {
		return query.LogPage{}, err
	}

	logs, err := query.ReadLogs(
		ctx,
		expr,
		queryString,
		jobLogPath,
		findingsLogPath,
		query.LogCursor(cursor),
		limit,
	)
	if err != nil {
		return query.LogPage{}, fmt.Errorf(
			"failed to read logs: %w",
			err,
		)
	}

	return logs, nil
}

func GetLog(ctx context.Context, cursor int64) (query.LogItem, error) {
	jobLogPath := os.Getenv("API_OBSERVER_JOB_LOG")

	if jobLogPath == "" {
		jobLogPath = "./logs/jobs.jsonl"
	}

	findingsLogPath := os.Getenv("API_OBSERVER_FINDINGS_LOG")

	if findingsLogPath == "" {
		findingsLogPath = "./logs/findings.jsonl"
	}

	log, err := query.ReadLog(
		ctx,
		jobLogPath,
		findingsLogPath,
		query.LogCursor(cursor),
	)
	if err != nil {
		return query.LogItem{}, fmt.Errorf(
			"failed to read log: %w",
			err,
		)
	}

	return log, err
}
