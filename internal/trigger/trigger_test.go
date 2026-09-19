package trigger

import (
	"errors"
	"os"
	"testing"

	conf "github.com/TechXploreLabs/seristack/internal/config"
)

func TestSingleStackCheck_ExistingStack(t *testing.T) {
	config := &conf.Config{
		Stacks: []conf.Stack{
			{
				Name:      "system-health",
				DependsOn: []string{"base-stack"},
			},
			{
				Name: "inspect-runtime",
			},
		},
	}

	stack := "system-health"

	got, err := SingleStackCheck(config, &stack)
	if err != nil {
		t.Fatalf("SingleStackCheck() unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("SingleStackCheck() returned nil config")
	}

	if len(got.Stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(got.Stacks))
	}

	if got.Stacks[0].Name != "system-health" {
		t.Errorf(
			"expected stack name %q, got %q",
			"system-health",
			got.Stacks[0].Name,
		)
	}

	if got.Stacks[0].DependsOn != nil {
		t.Errorf(
			"expected DependsOn to be nil, got %#v",
			got.Stacks[0].DependsOn,
		)
	}
}

func TestSingleStackCheck_NonExistingStack(t *testing.T) {
	config := &conf.Config{
		Stacks: []conf.Stack{
			{
				Name: "system-health",
			},
		},
	}

	stack := "does-not-exist"

	got, err := SingleStackCheck(config, &stack)

	if err == nil {
		t.Fatal("SingleStackCheck() expected error, got nil")
	}

	if got != nil {
		t.Errorf("expected nil config, got %#v", got)
	}

	expected := "Stack not exist."
	if err.Error() != expected {
		t.Errorf(
			"expected error %q, got %q",
			expected,
			err.Error(),
		)
	}
}

func TestSingleStackCheck_EmptyStackName(t *testing.T) {
	config := &conf.Config{
		Stacks: []conf.Stack{
			{
				Name: "system-health",
			},
		},
	}

	stack := ""

	got, err := SingleStackCheck(config, &stack)

	if err == nil {
		t.Fatal("SingleStackCheck() expected error, got nil")
	}

	if got != nil {
		t.Errorf("expected nil config, got %#v", got)
	}
}

func TestRunTrigger_DependencyResolutionFailure(t *testing.T) {
	config := &conf.Config{
		Stacks: []conf.Stack{
			{
				Name:      "stack-a",
				DependsOn: []string{"stack-b"},
			},
			{
				Name:      "stack-b",
				DependsOn: []string{"stack-a"},
			},
		},
	}

	output := "yaml"
	varsMap := map[string]string{}

	got, err := RunTrigger(config, &output, &varsMap)

	if err == nil {
		t.Fatal("RunTrigger() expected dependency resolution error, got nil")
	}

	if got != nil {
		t.Errorf("expected nil result, got %#v", got)
	}

	expectedPrefix := "dependency resolution failed due to"
	if len(err.Error()) < len(expectedPrefix) ||
		err.Error()[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf(
			"expected error to start with %q, got %q",
			expectedPrefix,
			err.Error(),
		)
	}
}

func TestRunTrigger_ValidConfiguration(t *testing.T) {
	config := &conf.Config{
		Stacks: []conf.Stack{
			{
				Name: "system-health",
			},
		},
	}

	varsMap := map[string]string{}

	tests := []struct {
		name   string
		output string
	}{
		{
			name:   "empty output",
			output: "",
		},
		{
			name:   "yaml output",
			output: "yaml",
		},
		{
			name:   "json output",
			output: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.output

			got, err := RunTrigger(
				config,
				&output,
				&varsMap,
			)

			if err != nil {
				t.Fatalf(
					"RunTrigger() unexpected error for output %q: %v",
					tt.output,
					err,
				)
			}

			if tt.output == "" && got != nil {
				t.Errorf(
					"expected nil result for empty output, got %#v",
					got,
				)
			}
		})
	}
}

func TestRunTrigger_UnsupportedOutput(t *testing.T) {
	config := &conf.Config{
		Stacks: []conf.Stack{
			{
				Name: "system-health",
			},
		},
	}

	output := "xml"
	varsMap := map[string]string{}

	got, err := RunTrigger(
		config,
		&output,
		&varsMap,
	)

	if err != nil {
		t.Fatalf(
			"RunTrigger() unexpected error for unsupported output: %v",
			err,
		)
	}

	if got != nil {
		t.Errorf(
			"expected nil result for unsupported output, got %#v",
			got,
		)
	}
}

// Keep errors imported if the project's config types evolve to expose
// wrapped/sentinel errors and this test file is extended.
var _ = errors.Is
var _ = os.ErrNotExist
