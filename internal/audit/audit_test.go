package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNew_CreatesAuditLog(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "logs", "audit.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	defer logger.Close()

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("expected audit log file to exist: %v", err)
	}

	if info.IsDir() {
		t.Fatal("expected audit log path to be a file, got directory")
	}
}

func TestNew_CreatesParentDirectories(t *testing.T) {
	tempDir := t.TempDir()

	logPath := filepath.Join(
		tempDir,
		"foo",
		"bar",
		"audit.log",
	)

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	defer logger.Close()

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("expected log file to exist: %v", err)
	}
}

func TestWrite_WritesJSONEntry(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	defer logger.Close()

	entry := Entry{
		RequestID: "req-123",
		Event:     "stack_executed",
		Stack:     "system-health",
		Path:      "/mcp",
		Method:    "POST",
		SourceIP:  "192.168.1.10",
		Identity: map[string]string{
			"X-Auth-Request-User":   "sudha",
			"X-Auth-Request-Groups": "platform-admin",
		},
		Vars: map[string]string{
			"ENV": "production",
		},
		Success:    true,
		DurationMs: 25,
		Output:     "system-health is good",
	}

	if err := logger.Write(entry); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	var got Entry

	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("audit log contains invalid JSON: %v", err)
	}

	if got.RequestID != entry.RequestID {
		t.Errorf(
			"RequestID = %q, want %q",
			got.RequestID,
			entry.RequestID,
		)
	}

	if got.Event != entry.Event {
		t.Errorf(
			"Event = %q, want %q",
			got.Event,
			entry.Event,
		)
	}

	if got.Stack != entry.Stack {
		t.Errorf(
			"Stack = %q, want %q",
			got.Stack,
			entry.Stack,
		)
	}

	if got.Path != entry.Path {
		t.Errorf(
			"Path = %q, want %q",
			got.Path,
			entry.Path,
		)
	}

	if got.Method != entry.Method {
		t.Errorf(
			"Method = %q, want %q",
			got.Method,
			entry.Method,
		)
	}

	if got.SourceIP != entry.SourceIP {
		t.Errorf(
			"SourceIP = %q, want %q",
			got.SourceIP,
			entry.SourceIP,
		)
	}

	if got.Success != entry.Success {
		t.Errorf(
			"Success = %v, want %v",
			got.Success,
			entry.Success,
		)
	}

	if got.DurationMs != entry.DurationMs {
		t.Errorf(
			"DurationMs = %d, want %d",
			got.DurationMs,
			entry.DurationMs,
		)
	}

	if got.Output != entry.Output {
		t.Errorf(
			"Output = %q, want %q",
			got.Output,
			entry.Output,
		)
	}

	if got.Identity["X-Auth-Request-User"] != "sudha" {
		t.Errorf("Identity user was not written correctly")
	}

	if got.Identity["X-Auth-Request-Groups"] != "platform-admin" {
		t.Errorf("Identity groups were not written correctly")
	}

	if got.Vars["ENV"] != "production" {
		t.Errorf("Vars were not written correctly")
	}

	if got.Timestamp == "" {
		t.Fatal("Timestamp should be populated by Write()")
	}

	if _, err := time.Parse(time.RFC3339, got.Timestamp); err != nil {
		t.Fatalf(
			"Timestamp is not valid RFC3339: %q: %v",
			got.Timestamp,
			err,
		)
	}
}

func TestWrite_DefaultEvent(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	defer logger.Close()

	entry := Entry{
		Stack:      "system-health",
		Path:       "/mcp",
		Method:     "POST",
		Success:    true,
		DurationMs: 10,
	}

	if err := logger.Write(entry); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	var got Entry

	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to decode audit entry: %v", err)
	}

	if got.Event != "stack_executed" {
		t.Errorf(
			"Event = %q, want %q",
			got.Event,
			"stack_executed",
		)
	}
}

func TestWrite_PreservesExplicitEvent(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	defer logger.Close()

	entry := Entry{
		Event:      "authentication_failed",
		Stack:      "system-health",
		Success:    false,
		DurationMs: 5,
	}

	if err := logger.Write(entry); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	var got Entry

	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to decode audit entry: %v", err)
	}

	if got.Event != "authentication_failed" {
		t.Errorf(
			"Event = %q, want %q",
			got.Event,
			"authentication_failed",
		)
	}
}

func TestWrite_OmitsOptionalFields(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	defer logger.Close()

	entry := Entry{
		Stack:      "system-health",
		Success:    true,
		DurationMs: 10,
	}

	if err := logger.Write(entry); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	line := strings.TrimSpace(string(data))

	var raw map[string]interface{}

	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	optionalFields := []string{
		"request_id",
		"source_ip",
		"identity",
		"vars",
		"output",
		"error",
	}

	for _, field := range optionalFields {
		if _, exists := raw[field]; exists {
			t.Errorf(
				"expected optional field %q to be omitted",
				field,
			)
		}
	}
}

func TestWrite_AppendsEntries(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	defer logger.Close()

	first := Entry{
		Stack:      "system-health",
		Success:    true,
		DurationMs: 10,
	}

	second := Entry{
		Stack:      "inspect-runtime",
		Success:    false,
		DurationMs: 20,
		Error:      "command failed",
	}

	if err := logger.Write(first); err != nil {
		t.Fatalf("first Write() failed: %v", err)
	}

	if err := logger.Write(second); err != nil {
		t.Fatalf("second Write() failed: %v", err)
	}

	file, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("failed to open audit log: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var entries []Entry

	for scanner.Scan() {
		var entry Entry

		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("invalid JSON entry: %v", err)
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("failed reading audit log: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf(
			"expected 2 audit entries, got %d",
			len(entries),
		)
	}

	if entries[0].Stack != "system-health" {
		t.Errorf(
			"first stack = %q, want %q",
			entries[0].Stack,
			"system-health",
		)
	}

	if entries[1].Stack != "inspect-runtime" {
		t.Errorf(
			"second stack = %q, want %q",
			entries[1].Stack,
			"inspect-runtime",
		)
	}
}

func TestWrite_ConcurrentWrites(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	defer logger.Close()

	const writers = 20

	var wg sync.WaitGroup
	wg.Add(writers)

	for i := 0; i < writers; i++ {
		go func(i int) {
			defer wg.Done()

			entry := Entry{
				RequestID:  "request-" + string(rune('A'+i)),
				Stack:      "system-health",
				Success:    true,
				DurationMs: int64(i),
			}

			if err := logger.Write(entry); err != nil {
				t.Errorf("Write() failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	file, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("failed to open audit log: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	count := 0

	for scanner.Scan() {
		var entry Entry

		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf(
				"invalid JSON at entry %d: %v",
				count,
				err,
			)
		}

		count++
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("failed reading audit log: %v", err)
	}

	if count != writers {
		t.Fatalf(
			"expected %d entries, got %d",
			writers,
			count,
		)
	}
}

func TestClose(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	if err := logger.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}
}

func TestWrite_AfterCloseReturnsError(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger, err := New(logPath)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	if err := logger.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	entry := Entry{
		Stack:   "system-health",
		Success: true,
	}

	if err := logger.Write(entry); err == nil {
		t.Fatal("Write() expected error after Close(), got nil")
	}
}
