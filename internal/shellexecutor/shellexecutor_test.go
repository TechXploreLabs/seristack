// Unit tests for Shellargs
package shellexecutor

import (
	"strings"
	"testing"
	"time"

	"github.com/TechXploreLabs/seristack/internal/config"
)

func TestShellargs(t *testing.T) {
	// Typical usage: known shell and arg
	shell, arg, err := Shellargs("/bin/bash", "-c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shell != "/bin/bash" || arg != "-c" {
		t.Errorf("expected /bin/bash with -c, got %q and %q", shell, arg)
	}

	// Edge: empty shell string
	shell, arg, err = Shellargs("", "")
	// Accept empty shell/arg unless implementation changes
	if err != nil {
		t.Errorf("did not expect error for empty shell/arg, got %v", err)
	}

	// Edge: non-empty shell, empty arg
	shell, arg, err = Shellargs("/bin/sh", "")
	if err != nil && arg != "" {
		t.Errorf("arg can be empty, should not error, got: %v", err)
	}

	// Edge: shell with spaces
	shell, arg, err = Shellargs("bash --login", "-c")
	if err != nil {
		t.Errorf("should not error on shell with spaces, got: %v", err)
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
		Shell:         "/bin/sh",
		ShellArg:      "-c",
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
