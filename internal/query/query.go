// Package query handles the querying operations for API Observer's dashboard.
package query

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"observer/internal/audit"
	"os"
)

type LogCursor int64

type LogPage struct {
	Items  []LogItem
	Cursor LogCursor
}

type LogItem struct {
	Job      audit.AuditJob
	Findings []audit.Finding
}

func ReadLogs(
	ctx context.Context,
	jobsPath,
	findingsPath string,
	cursor LogCursor,
	limit int,
) (LogPage, error) {
	logs, position, err := readJobLog(ctx, jobsPath, cursor, limit)
	if err != nil {
		return LogPage{}, err
	}

	items, err := findJobFindings(ctx, logs, findingsPath)
	if err != nil {
		return LogPage{}, err
	}

	return LogPage{
		Items:  items,
		Cursor: LogCursor(position),
	}, nil
}

func readJobLog(ctx context.Context, path string, cursor LogCursor, limit int) ([]audit.AuditJob, LogCursor, error) {
	file, err := os.Open(path)
	if err != nil {
		return []audit.AuditJob{}, 0, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return []audit.AuditJob{}, 0, err
	}

	position := int64(cursor)

	if cursor == 0 {
		position = stat.Size()
	}

	logs := make([]audit.AuditJob, 0, limit)

	for len(logs) < limit && position > 0 {
		select {
		case <-ctx.Done():
			return []audit.AuditJob{}, 0, ctx.Err()
		default:
		}

		lineEnd := position

		// If the byte immediately before position is a newline,
		// don't treat that newline as the previous line separator.
		searchFrom := lineEnd - 1

		var b [1]byte

		if searchFrom >= 0 {
			if _, err := file.ReadAt(b[:], searchFrom); err != nil {
				return []audit.AuditJob{}, 0, err
			}

			if b[0] == '\n' {
				searchFrom--
			}
		}

		lineStart := int64(0)

		for searchFrom >= 0 {
			if _, err := file.ReadAt(b[:], searchFrom); err != nil {
				return []audit.AuditJob{}, 0, err
			}

			if b[0] == '\n' {
				lineStart = searchFrom + 1
				break
			}

			searchFrom--
		}

		size := lineEnd - lineStart

		line := make([]byte, size)

		if _, err := file.ReadAt(line, lineStart); err != nil {
			return []audit.AuditJob{}, 0, err
		}

		position = lineStart

		line = bytes.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		var log audit.AuditJob

		if err := json.Unmarshal(line, &log); err != nil {
			continue
		}

		logs = append(logs, log)
	}

	return logs, LogCursor(position), nil
}

func findJobFindings(
	ctx context.Context,
	logs []audit.AuditJob,
	path string,
) ([]LogItem, error) {
	items := make([]LogItem, 0, len(logs))

	jobIndex := make(map[string]int, len(logs))

	for _, log := range logs {
		jobIndex[log.ID] = len(items)

		items = append(items, LogItem{
			Job:      log,
			Findings: make([]audit.Finding, 0),
		})
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, err
		}

		line = bytes.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		var finding audit.Finding

		if err := json.Unmarshal(line, &finding); err != nil {
			continue
		}

		index, ok := jobIndex[finding.JobID]
		if !ok {
			continue
		}

		items[index].Findings = append(
			items[index].Findings,
			finding,
		)
	}

	return items, nil
}
