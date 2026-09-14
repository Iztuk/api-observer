// Package audit handles processing and storing jobs and findings.
package audit

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Finding struct {
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

type Job struct {
	Type JobType

	Request  *RequestJob
	Response *ResponseJob
}

type JobType string

const (
	JobTypeRequest  JobType = "request"
	JobTypeResponse JobType = "response"
)

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

func (f Finding) Log(l *log.Logger) {
	finding, _ := json.Marshal(f)

	l.Println(finding)
}
