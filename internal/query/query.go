// Package query handles the querying operations for API Observer's dashboard.
package query

import (
	"bytes"
	"context"
	"encoding/json"
	"observer/internal/audit"
	"os"
)

type LogCursor int64

type LogPage struct {
	Items  []audit.AuditJob
	Cursor LogCursor
}

func ReadJobLog(
	ctx context.Context,
	path string,
	cursor LogCursor,
	limit int,
) (LogPage, error) {
	file, err := os.Open(path)
	if err != nil {
		return LogPage{}, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return LogPage{}, err
	}

	position := int64(cursor)

	if cursor == 0 {
		position = stat.Size()
	}

	logs := make([]audit.AuditJob, 0, limit)

	for len(logs) < limit && position > 0 {
		select {
		case <-ctx.Done():
			return LogPage{}, ctx.Err()
		default:
		}

		lineEnd := position

		// If the byte immediately before position is a newline,
		// don't treat that newline as the previous line separator.
		searchFrom := lineEnd - 1

		var b [1]byte

		if searchFrom >= 0 {
			if _, err := file.ReadAt(b[:], searchFrom); err != nil {
				return LogPage{}, err
			}

			if b[0] == '\n' {
				searchFrom--
			}
		}

		lineStart := int64(0)

		for searchFrom >= 0 {
			if _, err := file.ReadAt(b[:], searchFrom); err != nil {
				return LogPage{}, err
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
			return LogPage{}, err
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

	return LogPage{
		Items:  logs,
		Cursor: LogCursor(position),
	}, nil
}
