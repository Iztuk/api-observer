package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type JSONLogStore struct {
	mu   sync.RWMutex
	file *os.File
}

func NewJSONLogStore() (*JSONLogStore, error) {
	path := os.Getenv("API_OBSERVER_LOG_STORE")
	if path == "" {
		path = "/var/log/observer/audit.jsonl"
	}

	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf(
			"create audit log directory %q: %w",
			dir,
			err,
		)
	}

	f, err := os.OpenFile(
		path,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0o640,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"open audit log file %q: %w",
			path,
			err,
		)
	}

	return &JSONLogStore{
		file: f,
	}, nil
}

func (s *JSONLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		return nil
	}

	if err := s.file.Close(); err != nil {
		return fmt.Errorf("close audit log file: %w", err)
	}

	s.file = nil
	return nil
}

func (s *JSONLogStore) SaveAuditResult(findings []Finding) error {
	if len(findings) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, finding := range findings {
		data, err := json.Marshal(finding)
		if err != nil {
			return fmt.Errorf("marshal audit finding: %w", err)
		}

		if _, err := s.file.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("write audit finding: %w", err)
		}
	}

	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("sync audit log: %w", err)
	}

	return nil
}
