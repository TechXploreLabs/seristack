package opsyseristackaction

import (
	"strings"
	"testing"
	"time"

	"github.com/TechXploreLabs/seristack/internal/config"
)

func TestNewConfigFromYAML_Valid(t *testing.T) {
	data := []byte(`
stacks:
  - name: system-health
    cmds:
      - echo "system-health is good"
`)

	got, err := NewConfigFromYAML(data)
	if err != nil {
		t.Fatalf(
			"NewConfigFromYAML() unexpected error: %v",
			err,
		)
	}

	if got == nil {
		t.Fatal("NewConfigFromYAML() returned nil config")
	}

	if len(got.Stacks) != 1 {
		t.Fatalf(
			"expected 1 stack, got %d",
			len(got.Stacks),
		)
	}

	stack := got.Stacks[0]

	if stack.Name != "system-health" {
		t.Errorf(
			"Name = %q, want %q",
			stack.Name,
			"system-health",
		)
	}

	if len(stack.Cmds) != 1 {
		t.Fatalf(
			"expected 1 command, got %d",
			len(stack.Cmds),
		)
	}

	if stack.Cmds[0] != `echo "system-health is good"` {
		t.Errorf(
			"Cmds[0] = %q, want %q",
			stack.Cmds[0],
			`echo "system-health is good"`,
		)
	}
}

func TestNewConfigFromYAML_InvalidYAML(t *testing.T) {
	data := []byte(`
stacks:
  - name: system-health
    cmds: [invalid
`)

	got, err := NewConfigFromYAML(data)

	if err == nil {
		t.Fatal(
			"NewConfigFromYAML() expected error, got nil",
		)
	}

	if got == nil {
		t.Fatal(
			"NewConfigFromYAML() should return a non-nil empty config",
		)
	}

	if len(got.Stacks) != 0 {
		t.Errorf(
			"expected empty config, got %d stacks",
			len(got.Stacks),
		)
	}

	if !strings.Contains(err.Error(), "failed to decode stack config") {
		t.Errorf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestNewConfigFromYAML_EmptyDocument(t *testing.T) {
	data := []byte("")

	got, err := NewConfigFromYAML(data)

	if err != nil {
		t.Fatalf(
			"NewConfigFromYAML() unexpected error: %v",
			err,
		)
	}

	if got == nil {
		t.Fatal("expected non-nil config")
	}

	if len(got.Stacks) != 0 {
		t.Errorf(
			"expected 0 stacks, got %d",
			len(got.Stacks),
		)
	}
}

func TestNewConfigFromYAML_MultipleStacks(t *testing.T) {
	data := []byte(`
stacks:
  - name: system-health
    cmds:
      - echo "system-health is good"

  - name: inspect-runtime
    cmds:
      - echo "runtime is healthy"

  - name: production-operation
    cmds:
      - echo "production operation"
`)

	got, err := NewConfigFromYAML(data)

	if err != nil {
		t.Fatalf(
			"NewConfigFromYAML() unexpected error: %v",
			err,
		)
	}

	if len(got.Stacks) != 3 {
		t.Fatalf(
			"expected 3 stacks, got %d",
			len(got.Stacks),
		)
	}

	expectedNames := []string{
		"system-health",
		"inspect-runtime",
		"production-operation",
	}

	for i, expected := range expectedNames {
		if got.Stacks[i].Name != expected {
			t.Errorf(
				"stack[%d].Name = %q, want %q",
				i,
				got.Stacks[i].Name,
				expected,
			)
		}

		if len(got.Stacks[i].Cmds) != 1 {
			t.Errorf(
				"stack[%d] expected 1 command, got %d",
				i,
				len(got.Stacks[i].Cmds),
			)
		}
	}
}

func TestNewConfigFromYAML_StackFields(t *testing.T) {
	data := []byte(`
stacks:
  - name: system-health
    description: System health check
    method: POST
    urlPath: /health
    workDir: /tmp
    continueOnError: true
    dependsOn:
      - base-stack
    matchAccess: ALL
    access:
      - headerName: X-Auth-Request-Groups
        headerValue:
          - platform
          - platform-admin
    executionMode: sequential
    count: 2
    shell: bash
    shellArg: -c
    cmds:
      - echo "health"
    timeouts: 30s
    output: yaml
    discardOutput:
      - debug
`)

	got, err := NewConfigFromYAML(data)

	if err != nil {
		t.Fatalf(
			"NewConfigFromYAML() unexpected error: %v",
			err,
		)
	}

	if len(got.Stacks) != 1 {
		t.Fatalf(
			"expected 1 stack, got %d",
			len(got.Stacks),
		)
	}

	stack := got.Stacks[0]

	if stack.Name != "system-health" {
		t.Errorf("Name = %q", stack.Name)
	}

	if stack.Description != "System health check" {
		t.Errorf("Description = %q", stack.Description)
	}

	if stack.Method != "POST" {
		t.Errorf("Method = %q", stack.Method)
	}

	if stack.UrlPath != "/health" {
		t.Errorf("UrlPath = %q", stack.UrlPath)
	}

	if stack.WorkDir != "/tmp" {
		t.Errorf("WorkDir = %q", stack.WorkDir)
	}

	if !stack.ContinueOnError {
		t.Error("expected ContinueOnError to be true")
	}

	if len(stack.DependsOn) != 1 ||
		stack.DependsOn[0] != "base-stack" {
		t.Errorf(
			"DependsOn = %#v",
			stack.DependsOn,
		)
	}

	if stack.MatchAccess != "ALL" {
		t.Errorf(
			"MatchAccess = %q",
			stack.MatchAccess,
		)
	}

	if len(stack.Access) != 1 {
		t.Fatalf(
			"expected 1 access rule, got %d",
			len(stack.Access),
		)
	}

	if stack.Access[0].HeaderName != "X-Auth-Request-Groups" {
		t.Errorf(
			"HeaderName = %q",
			stack.Access[0].HeaderName,
		)
	}

	if len(stack.Access[0].HeaderValue) != 2 {
		t.Fatalf(
			"expected 2 header values, got %d",
			len(stack.Access[0].HeaderValue),
		)
	}

	if stack.ExecutionMode != "sequential" {
		t.Errorf(
			"ExecutionMode = %q",
			stack.ExecutionMode,
		)
	}

	if stack.Count != 2 {
		t.Errorf(
			"Count = %d, want 2",
			stack.Count,
		)
	}

	if stack.Shell != "bash" {
		t.Errorf(
			"Shell = %q",
			stack.Shell,
		)
	}

	if stack.ShellArg != "-c" {
		t.Errorf(
			"ShellArg = %q",
			stack.ShellArg,
		)
	}

	if len(stack.Cmds) != 1 {
		t.Fatalf(
			"expected 1 command, got %d",
			len(stack.Cmds),
		)
	}

	if stack.Timeouts != "30s" {
		t.Errorf(
			"Timeouts = %q",
			stack.Timeouts,
		)
	}

	if stack.Output != "yaml" {
		t.Errorf(
			"Output = %q",
			stack.Output,
		)
	}

	if len(stack.DiscardOutput) != 1 ||
		stack.DiscardOutput[0] != "debug" {
		t.Errorf(
			"DiscardOutput = %#v",
			stack.DiscardOutput,
		)
	}
}

func TestNewConfigFromYAML_NormalizesVariables(t *testing.T) {
	data := []byte(`
stacks:
  - name: system-health
    cmds:
      - echo "${ENV}"
    vars:
      - name: ENV
        value: production
`)

	got, err := NewConfigFromYAML(data)

	if err != nil {
		t.Fatalf(
			"NewConfigFromYAML() unexpected error: %v",
			err,
		)
	}

	if len(got.Stacks) != 1 {
		t.Fatalf(
			"expected 1 stack, got %d",
			len(got.Stacks),
		)
	}

	stack := got.Stacks[0]

	if len(stack.Variables) != 1 {
		t.Fatalf(
			"expected 1 variable definition, got %d",
			len(stack.Variables),
		)
	}

	if stack.Variables[0].Name != "ENV" {
		t.Errorf(
			"variable name = %q, want %q",
			stack.Variables[0].Name,
			"ENV",
		)
	}

	if stack.Variables[0].Value != "production" {
		t.Errorf(
			"variable value = %q, want %q",
			stack.Variables[0].Value,
			"production",
		)
	}

	if stack.Vars == nil {
		t.Fatal("expected normalized Vars map to be initialized")
	}

	if stack.Vars["ENV"] != "production" {
		t.Errorf(
			"Vars[ENV] = %q, want %q",
			stack.Vars["ENV"],
			"production",
		)
	}
}

func TestNewConfigFromYAML_NormalizesTimeout(t *testing.T) {
	data := []byte(`
stacks:
  - name: system-health
    cmds:
      - echo "ok"
    timeouts: 30s
`)

	got, err := NewConfigFromYAML(data)

	if err != nil {
		t.Fatalf(
			"NewConfigFromYAML() unexpected error: %v",
			err,
		)
	}

	if len(got.Stacks) != 1 {
		t.Fatalf(
			"expected 1 stack, got %d",
			len(got.Stacks),
		)
	}

	if got.Stacks[0].Timeouts != "30s" {
		t.Errorf(
			"Timeouts = %q, want %q",
			got.Stacks[0].Timeouts,
			"30s",
		)
	}
}

func TestOpsySeristack_InvalidStack(t *testing.T) {
	conf := &Config{
		Config: &config.Config{
			Stacks: []config.Stack{
				{
					Name: "system-health",
					Cmds: []string{
						`echo "ok"`,
					},
				},
			},
		},
		StackName: "does-not-exist",
		Vars:      map[string]string{},
		Format:    "yaml",
	}

	_, err := OpsySeristack(conf)

	if err == nil {
		t.Fatal(
			"OpsySeristack() expected error, got nil",
		)
	}

	if !strings.Contains(err.Error(), "not exist") &&
		!strings.Contains(err.Error(), "does not exist") {
		t.Errorf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestOpsySeristack_ValidStack(t *testing.T) {
	conf := &Config{
		Config: &config.Config{
			Stacks: []config.Stack{
				{
					Name: "system-health",
					Cmds: []string{
						`echo "system-health is good"`,
					},
				},
			},
		},
		StackName: "system-health",
		Vars:      map[string]string{},
		Format:    "yaml",
	}

	result, err := OpsySeristack(conf)

	if err != nil {
		t.Fatalf(
			"OpsySeristack() unexpected error: %v",
			err,
		)
	}

	if result.Name != "system-health" {
		t.Errorf(
			"Name = %q, want %q",
			result.Name,
			"system-health",
		)
	}

	if !result.Success {
		t.Fatalf(
			"stack execution failed: success=%v output=%q error=%q",
			result.Success,
			result.Output,
			result.Error,
		)
	}

	if result.Duration < 0 {
		t.Errorf(
			"Duration = %v, should not be negative",
			result.Duration,
		)
	}
}

func TestResultFields(t *testing.T) {
	result := Result{
		Name:            "system-health",
		Success:         true,
		Output:          "ok",
		Error:           "",
		Duration:        100 * time.Millisecond,
		ContinueOnError: false,
	}

	if result.Name != "system-health" {
		t.Errorf(
			"unexpected Name: %q",
			result.Name,
		)
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.Output != "ok" {
		t.Errorf(
			"unexpected Output: %q",
			result.Output,
		)
	}

	if result.Duration != 100*time.Millisecond {
		t.Errorf(
			"Duration = %v, want %v",
			result.Duration,
			100*time.Millisecond,
		)
	}

	if result.ContinueOnError {
		t.Error(
			"expected ContinueOnError to be false",
		)
	}
}
