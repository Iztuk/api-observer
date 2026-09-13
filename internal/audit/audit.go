// Package audit handles processing and storing jobs and findings.
package audit

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Findings struct {
	ID       uuid.UUID
	Title    string
	Message  string
	Severity string
	Tags     []string

	Metadata  Metadata
	Timestamp time.Time
}

type Metadata struct {
	RequestID string
	Source    string
}

type RequestJob struct {
	Method        string
	URL           string
	Header        http.Header
	Body          string
	ContentLength int64

	Metadata Metadata
}

type ResponseJob struct {
	StatusCode    int
	Header        http.Header
	Body          string
	ContentLength int64

	Metadata Metadata
}
