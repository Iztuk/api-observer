package dashboard

import (
	"context"
	"fmt"
	"observer/internal/audit"
	"observer/internal/query"
	"os"
	"time"
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

func GetAnalysisCursor(ctx context.Context, timeTo time.Time) (query.LogCursor, error) {
	jobLogPath := os.Getenv("API_OBSERVER_JOB_LOG")

	if jobLogPath == "" {
		jobLogPath = "./logs/jobs.jsonl"
	}

	cursor, err := query.FindToCursor(ctx, jobLogPath, timeTo)
	if err != nil {
		return -1, err
	}

	return cursor, nil
}

func GetAnalysisLogs(ctx context.Context, queryString, analysisRules string, cursor query.LogCursor, timeFrom, timeTo time.Time) (query.LogPage, time.Time, error) {
	var logPage query.LogPage

	jobLogPath := os.Getenv("API_OBSERVER_JOB_LOG")

	if jobLogPath == "" {
		jobLogPath = "./logs/jobs.jsonl"
	}

	rules, err := audit.ParseHostRules(analysisRules)
	if err != nil {
		return query.LogPage{}, time.Time{}, err
	}

	if cursor == -1 {
		cursor, err = query.FindToCursor(ctx, jobLogPath, timeTo)
		if err != nil {
			return query.LogPage{}, time.Time{}, fmt.Errorf(
				"failed to identify 'To' cursor: %w",
				err,
			)
		}
	}

	expr, err := query.ParseQuery(queryString)
	if err != nil {
		return query.LogPage{}, time.Time{}, err
	}

	registry := audit.NewContractRegistry()
	engine := audit.NewRuleEngine(registry)

	logItems := make([]query.LogItem, 0)
	for len(logItems) < 25 {
		items, curr, end, err := query.ReadJobLogs(ctx, jobLogPath, timeFrom, cursor, 25-len(logItems))
		if err != nil {
			return query.LogPage{}, time.Time{}, fmt.Errorf(
				"failed to read log: %w",
				err,
			)
		}

		for i, item := range items {
			host := item.Job.Host
			if !registry.HostExists(host) {
				registry.RegisterHost(host, nil, rules)
			}

			job, err := audit.JobFromAuditJob(item.Job)
			if err != nil {
				return query.LogPage{}, time.Time{}, err
			}

			findings, err := engine.Evaluate(job, item.Job.ID)
			if err != nil {
				return query.LogPage{}, time.Time{}, err
			}

			items[i].Findings = append(items[i].Findings, findings...)

			// Query evaluation
			valid, err := query.EvaluateExpression(queryString, expr, items[i])
			if err != nil {
				return query.LogPage{}, time.Time{}, err
			}

			if valid {
				logItems = append(logItems, items[i])
			}
		}

		cursor = curr

		// Case where there are less than 25 logs
		if end {
			break
		}
	}

	logPage.Items = logItems
	logPage.Cursor = cursor

	return logPage, timeFrom, nil
}

func GetAnalysisLog(ctx context.Context, analysisRules string, cursor int64) (query.LogItem, error) {
	jobLogPath := os.Getenv("API_OBSERVER_JOB_LOG")

	if jobLogPath == "" {
		jobLogPath = "./logs/jobs.jsonl"
	}

	rules, err := audit.ParseHostRules(analysisRules)
	if err != nil {
		return query.LogItem{}, err
	}

	log, err := query.ReadJobLog(
		ctx,
		jobLogPath,
		query.LogCursor(cursor),
	)
	if err != nil {
		return query.LogItem{}, fmt.Errorf(
			"failed to read log: %w",
			err,
		)
	}

	registry := audit.NewContractRegistry()

	registry.RegisterHost(log.Job.Host, nil, rules)

	engine := audit.NewRuleEngine(registry)

	job, err := audit.JobFromAuditJob(log.Job)
	if err != nil {
		return query.LogItem{}, err
	}

	findings, err := engine.Evaluate(job, log.Job.ID)
	if err != nil {
		return query.LogItem{}, err
	}

	log.Findings = append(log.Findings, findings...)

	return log, err
}

func ParseDateTimeInput(value string) (time.Time, error) {
	const layout = "2006-01-02T15:04"

	t, err := time.ParseInLocation(layout, value, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"invalid datetime %q: %w",
			value,
			err,
		)
	}

	return t, nil
}
