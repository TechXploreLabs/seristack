// Unit tests for Shellargs
package shellexecutor

import (
	"strings"
	"testing"
	"time"

	"github.com/TechXploreLabs/seristack/internal/config"
)

func TestParallelNoRace(t *testing.T) {
	e := &config.Executor{SourceDir: t.TempDir()}
	s := &config.Stack{
		Name:          "t",
		Count:         5,
		ExecutionMode: "PARALLEL",
		Cmds:          []string{"echo a", "echo b", "echo c"},
		Vars:          map[string]string{},
	}
	r := ExecuteShell(e, s)
	if !r.Success {
		t.Fatal(r.Error)
	}
}

func TestCommandTimeoutDuration(t *testing.T) {
	t.Run("default timeout", func(t *testing.T) {
		duration, err := commandTimeoutDuration("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if duration != config.DefaultCommandTimeout {
			t.Fatalf("expected default timeout %s, got %s", config.DefaultCommandTimeout, duration)
		}
	})

	t.Run("custom timeout", func(t *testing.T) {
		duration, err := commandTimeoutDuration("2s")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if duration != 2*time.Second {
			t.Fatalf("expected 2s, got %s", duration)
		}
	})

	t.Run("invalid timeout", func(t *testing.T) {
		_, err := commandTimeoutDuration("invalid")
		if err == nil {
			t.Fatal("expected error for invalid timeout")
		}
	})

	t.Run("zero timeout", func(t *testing.T) {
		_, err := commandTimeoutDuration("0s")
		if err == nil {
			t.Fatal("expected error for zero timeout")
		}
	})
}

func TestExecuteShellCommandTimeout(t *testing.T) {
	result := ExecuteShell(&config.Executor{SourceDir: t.TempDir()}, &config.Stack{
		Name:          "timeout-stack",
		Count:         1,
		ExecutionMode: "SEQUENTIAL",
		Cmds:          []string{"sleep 2"},
		Timeouts:      "100ms",
	})

	if result.Success {
		t.Fatal("expected timeout to fail stack execution")
	}
	if !strings.Contains(result.Error, "command timed out after") {
		t.Fatalf("expected timeout error, got: %q", result.Error)
	}
}
