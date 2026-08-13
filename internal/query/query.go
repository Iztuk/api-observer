// Package query handles the querying operations for API Observer's dashboard.
package query

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	Job       audit.AuditJob
	JobCursor LogCursor
	Findings  []audit.Finding
}

func ReadLogs(
	ctx context.Context,
	jobsPath,
	findingsPath string,
	cursor LogCursor,
	limit int,
) (LogPage, error) {
	items, position, err := readJobLog(
		ctx,
		jobsPath,
		cursor,
		limit,
	)
	if err != nil {
		return LogPage{}, err
	}

	items, err = findJobFindings(
		ctx,
		items,
		findingsPath,
	)
	if err != nil {
		return LogPage{}, err
	}

	return LogPage{
		Items:  items,
		Cursor: position,
	}, nil
}

func ReadLog(
	ctx context.Context,
	jobsPath,
	findingsPath string,
	cursor LogCursor,
) (LogItem, error) {
	item, err := readJobLogForward(
		ctx,
		jobsPath,
		cursor,
	)
	if err != nil {
		return LogItem{}, err
	}

	items, err := findJobFindings(
		ctx,
		[]LogItem{item},
		findingsPath,
	)
	if err != nil {
		return LogItem{}, err
	}

	if len(items) == 0 {
		return LogItem{}, fmt.Errorf(
			"log not found at cursor %d",
			cursor,
		)
	}

	return items[0], nil
}

func readJobLog(
	ctx context.Context,
	path string,
	cursor LogCursor,
	limit int,
) ([]LogItem, LogCursor, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}

	position := int64(cursor)

	if cursor == 0 {
		position = stat.Size()
	}

	items := make([]LogItem, 0, limit)

	for len(items) < limit && position > 0 {
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		default:
		}

		lineEnd := position
		searchFrom := lineEnd - 1

		var b [1]byte

		if searchFrom >= 0 {
			if _, err := file.ReadAt(b[:], searchFrom); err != nil {
				return nil, 0, err
			}

			if b[0] == '\n' {
				searchFrom--
			}
		}

		lineStart := int64(0)

		for searchFrom >= 0 {
			if _, err := file.ReadAt(b[:], searchFrom); err != nil {
				return nil, 0, err
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
			return nil, 0, err
		}

		// This becomes the starting point for the next older record.
		position = lineStart

		line = bytes.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		var job audit.AuditJob

		if err := json.Unmarshal(line, &job); err != nil {
			continue
		}

		items = append(items, LogItem{
			Job:       job,
			JobCursor: LogCursor(lineStart),
			Findings:  make([]audit.Finding, 0),
		})
	}

	return items, LogCursor(position), nil
}

func readJobLogForward(
	ctx context.Context,
	path string,
	cursor LogCursor,
) (LogItem, error) {
	file, err := os.Open(path)
	if err != nil {
		return LogItem{}, err
	}
	defer file.Close()

	if _, err := file.Seek(int64(cursor), io.SeekStart); err != nil {
		return LogItem{}, err
	}

	reader := bufio.NewReader(file)

	select {
	case <-ctx.Done():
		return LogItem{}, ctx.Err()
	default:
	}

	line, err := reader.ReadBytes('\n')

	if err != nil && !errors.Is(err, io.EOF) {
		return LogItem{}, err
	}

	line = bytes.TrimSpace(line)

	if len(line) == 0 {
		return LogItem{}, fmt.Errorf(
			"no log found at cursor %d",
			cursor,
		)
	}

	var job audit.AuditJob

	if err := json.Unmarshal(line, &job); err != nil {
		return LogItem{}, fmt.Errorf(
			"failed to decode log at cursor %d: %w",
			cursor,
			err,
		)
	}

	return LogItem{
		Job:       job,
		JobCursor: cursor,
		Findings:  make([]audit.Finding, 0),
	}, nil
}

func findJobFindings(
	ctx context.Context,
	items []LogItem,
	path string,
) ([]LogItem, error) {
	jobIndex := make(map[string]int, len(items))

	for i, item := range items {
		jobIndex[item.Job.ID] = i
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

func parseQuery(rawString string) (Expression, error) {
	l := Lexer{
		Query:    rawString,
		Position: 0,
		Tokens:   make([]Token, 0),
	}

	err := l.Process()
	if err != nil {
		return nil, err
	}

	p := Parser{
		Query:         rawString,
		Tokens:        l.Tokens,
		TokenPosition: 0,
	}

	expr, err := p.Parse()
	if err != nil {
		return nil, err
	}

	return expr, nil
}
