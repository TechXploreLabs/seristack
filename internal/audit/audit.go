package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	Timestamp  string            `json:"timestamp"`
	RequestID  string            `json:"request_id,omitempty"`
	Event      string            `json:"event"`
	Stack      string            `json:"stack"`
	Path       string            `json:"path"`
	Method     string            `json:"method"`
	SourceIP   string            `json:"source_ip,omitempty"`
	Identity   map[string]string `json:"identity,omitempty"`
	Vars       map[string]string `json:"vars,omitempty"`
	Success    bool              `json:"success"`
	DurationMs int64             `json:"duration_ms"`
	Error      string            `json:"error,omitempty"`
}

type Logger struct {
	mu   sync.Mutex
	file *os.File
}

func New(path string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600) //nolint:gosec // G304: path is supplied by the operator via --audit-log CLI flag, not user input
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}
	return &Logger{file: f}, nil
}

func (l *Logger) Write(entry Entry) error {
	entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	if entry.Event == "" {
		entry.Event = "stack_executed"
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	_, err = l.file.Write(append(data, '\n'))
	return err
}

func (l *Logger) Close() error {
	return l.file.Close()
}
