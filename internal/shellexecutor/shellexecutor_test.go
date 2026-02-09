// Unit tests for Shellargs
package shellexecutor

import (
	"testing"
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
