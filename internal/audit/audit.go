// Package audit handles processing and storing jobs and findings.
package audit

import (
	"time"

	"github.com/google/uuid"
)

type Findings struct {
	ID       uuid.UUID
	Title    string
	Message  string
	Severity string
	Tags     []string

	Metadata  any
	Timestamp time.Time
}

type RequestJob struct {
}

type ResponseJob struct {
}
