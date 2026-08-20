package audit

import (
	"fmt"
	"time"
)

func JobFromAuditJob(record AuditJob) (Job, error) {
	ts, err := time.Parse(time.RFC3339Nano, record.Timestamp)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse timestamp: %s",
			record.Timestamp,
		)
	}

	switch record.Type {
	case string(RequestJobType):
		return &RequestJob{
			Type: JobType(record.Type),
			Meta: Metadata{
				RequestID: record.RequestID,
				Host:      record.Host,
				Method:    record.Method,
				Path:      record.Path,
				Query:     record.Query,
				Status:    record.Status,
				Timestamp: ts,
			},
			Headers: record.Headers,
			Body:    []byte(record.Body),
		}, nil

	case string(ResponseJobType):
		return &ResponseJob{
			Type: JobType(record.Type),
			Meta: Metadata{
				RequestID: record.RequestID,
				Host:      record.Host,
				Method:    record.Method,
				Path:      record.Path,
				Query:     record.Query,
				Status:    record.Status,
				Timestamp: ts,
			},
			Headers: record.Headers,
			Body:    []byte(record.Body),
		}, nil

	case string(FailureJobType):
		return &FailureJob{
			Type: JobType(record.Type),
			Meta: Metadata{
				RequestID: record.RequestID,
				Host:      record.Host,
				Method:    record.Method,
				Path:      record.Path,
				Query:     record.Query,
				Status:    record.Status,
				Timestamp: ts,
			},
			Error: record.Error,
		}, nil

	default:
		return nil, fmt.Errorf(
			"unsupported audit job type: %q",
			record.Type,
		)
	}
}
