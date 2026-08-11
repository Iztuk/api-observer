package audit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type JSONLogStore struct {
	jobMu   sync.RWMutex
	jobFile *os.File

	auditMu   sync.RWMutex
	auditFile *os.File
}

type Finding struct {
	ID        string
	JobID     string
	RuleID    string
	Title     string
	Message   string
	CreatedAt time.Time
}

type AuditJob struct {
	ID   string `json:"id"`
	Type string `json:"type"`

	RequestID string `json:"request_id"`
	Host      string `json:"host"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Query     string `json:"query,omitempty"`
	Status    int    `json:"status,omitempty"`
	Timestamp string `json:"timestamp"`

	Headers http.Header `json:"headers,omitempty"`
	Body    string      `json:"body,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func NewJSONLogStore() (*JSONLogStore, error) {
	jobPath := os.Getenv("API_OBSERVER_JOB_LOG")
	if jobPath == "" {
		err := os.Setenv("API_OBSERVER_JOB_LOG", "./logs/jobs.jsonl")
		if err != nil {
			return nil, fmt.Errorf("failed to set API_OBSERVER_JOB_LOG environment variable to ./logs/jobs.jsonl")
		}
		jobPath = os.Getenv("API_OBSERVER_JOB_LOG")
	}

	auditPath := os.Getenv("API_OBSERVER_FINDINGS_LOG")
	if auditPath == "" {
		err := os.Setenv("API_OBSERVER_FINDINGS_LOG", "./logs/findings.jsonl")
		if err != nil {
			return nil, fmt.Errorf("failed to set API_OBSERVER_JOB_LOG environment variable to ./logs/findings.jsonl")
		}
		auditPath = os.Getenv("API_OBSERVER_FINDINGS_LOG")
	}

	dir := filepath.Dir(jobPath)

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf(
			"create audit log directory %q: %w",
			dir,
			err,
		)
	}

	jf, err := os.OpenFile(
		jobPath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0o640,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"open job log file %q: %w",
			jobPath,
			err,
		)
	}

	af, err := os.OpenFile(
		auditPath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0o640,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"open audit log file %q: %w",
			auditPath,
			err,
		)
	}

	return &JSONLogStore{
		jobFile:   jf,
		auditFile: af,
	}, nil
}

func (s *JSONLogStore) Close() error {
	s.jobMu.Lock()
	s.auditMu.Lock()
	defer s.jobMu.Unlock()
	defer s.auditMu.Unlock()

	if s.auditFile == nil && s.jobFile == nil {
		return nil
	}

	if err := s.jobFile.Close(); err != nil {
		return fmt.Errorf("close job log file: %w", err)
	}

	if err := s.auditFile.Close(); err != nil {
		return fmt.Errorf("close audit log file: %w", err)
	}

	s.jobFile = nil
	s.auditFile = nil
	return nil
}

func (s *JSONLogStore) SaveJob(job Job, jobID string) error {
	if job == nil {
		return fmt.Errorf("missing job")
	}

	if jobID == "" {
		return fmt.Errorf("missing job ID")

	}

	var (
		jobType string
		meta    Metadata
		headers http.Header
		body    string
		errStr  string
	)

	switch j := job.(type) {
	case *RequestJob:
		jobType = string(j.JobType())
		meta = j.Meta

		headers = j.Headers

		body = string(j.Body)
	case *ResponseJob:
		jobType = string(j.JobType())
		meta = j.Meta

		headers = j.Headers

		body = string(j.Body)
	case *FailureJob:
		jobType = string(j.JobType())
		meta = j.Meta

		errStr = j.Error
	default:
		return fmt.Errorf("unknown job type: %T", job)
	}

	auditJob := AuditJob{
		ID:        jobID,
		Type:      jobType,
		RequestID: meta.RequestID,
		Host:      meta.Host,
		Method:    meta.Method,
		Path:      meta.Path,
		Query:     meta.Query,
		Status:    meta.Status,
		Timestamp: meta.Timestamp.Format(time.RFC3339Nano),
		Headers:   headers,
		Body:      body,
		Error:     errStr,
	}

	s.jobMu.Lock()
	defer s.jobMu.Unlock()

	data, err := json.Marshal(auditJob)
	if err != nil {
		return fmt.Errorf("marshal audit job: %w", err)
	}

	if _, err := s.jobFile.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write audit job: %w", err)
	}

	if err := s.jobFile.Sync(); err != nil {
		return fmt.Errorf("sync audit job log: %w", err)
	}

	return nil
}

func (s *JSONLogStore) SaveAuditResult(findings []Finding) error {
	if len(findings) == 0 {
		return nil
	}

	s.auditMu.Lock()
	defer s.auditMu.Unlock()

	for _, finding := range findings {
		data, err := json.Marshal(finding)
		if err != nil {
			return fmt.Errorf("marshal audit finding: %w", err)
		}

		if _, err := s.auditFile.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("write audit finding: %w", err)
		}
	}

	if err := s.auditFile.Sync(); err != nil {
		return fmt.Errorf("sync audit log: %w", err)
	}

	return nil
}

func newUUID() string {
	return uuid.NewString()
}

func marshalHeaders(h http.Header) (string, error) {
	jsonData, err := json.Marshal(h)
	return string(jsonData), err
}
