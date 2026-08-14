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
	expr Expression,
	queryString,
	jobsPath,
	findingsPath string,
	cursor LogCursor,
	limit int,
) (LogPage, error) {
	items, position, err := readJobLog(
		ctx,
		expr,
		queryString,
		jobsPath,
		findingsPath,
		cursor,
		limit,
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
		findingsPath,
		cursor,
	)
	if err != nil {
		return LogItem{}, err
	}

	return item, nil
}

func readJobLog(
	ctx context.Context,
	expr Expression,
	queryString,
	jobPath,
	findingsPath string,
	cursor LogCursor,
	limit int,
) ([]LogItem, LogCursor, error) {
	file, err := os.Open(jobPath)
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

		item := LogItem{
			Job:       job,
			JobCursor: LogCursor(lineStart),
			Findings:  make([]audit.Finding, 0),
		}

		item, err = findJobFindings(ctx, item, findingsPath)
		if err != nil {
			return nil, 0, err
		}

		valid, err := evaluateExpression(queryString, expr, item)
		if err != nil {
			return nil, 0, err
		}

		if valid {
			items = append(items, item)
		}
	}

	return items, LogCursor(position), nil
}

func readJobLogForward(
	ctx context.Context,
	jobPath,
	findingsPath string,
	cursor LogCursor,
) (LogItem, error) {
	file, err := os.Open(jobPath)
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

	item := LogItem{
		Job:       job,
		JobCursor: cursor,
		Findings:  make([]audit.Finding, 0),
	}

	item, err = findJobFindings(ctx, item, findingsPath)
	if err != nil {
		return LogItem{}, err
	}

	return item, nil
}

func findJobFindings(
	ctx context.Context,
	item LogItem,
	path string,
) (LogItem, error) {
	file, err := os.Open(path)
	if err != nil {
		return LogItem{}, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		select {
		case <-ctx.Done():
			return LogItem{}, ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return LogItem{}, err
		}

		line = bytes.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		var finding audit.Finding

		if err := json.Unmarshal(line, &finding); err != nil {
			continue
		}

		if item.Job.ID == finding.JobID {
			item.Findings = append(item.Findings, finding)
		}
	}

	return item, nil
}

func ParseQuery(rawString string) (Expression, error) {
	if len(rawString) == 0 {
		return nil, nil
	}

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
