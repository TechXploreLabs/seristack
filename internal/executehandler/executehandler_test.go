package executehandler

import (
	"strings"
	"testing"

	"github.com/TechXploreLabs/seristack/internal/config"
)

func TestStackmap(t *testing.T) {
	stacks := []config.Stack{
		{Name: "stack1"},
		{Name: "stack2"},
		{Name: "stack3"},
	}

	got := Stackmap(stacks)

	if len(got) != len(stacks) {
		t.Fatalf("Stackmap() returned %d stacks, want %d", len(got), len(stacks))
	}

	for _, stack := range stacks {
		gotStack, ok := got[stack.Name]
		if !ok {
			t.Fatalf("Stackmap() missing stack %q", stack.Name)
		}

		if gotStack.Name != stack.Name {
			t.Errorf("Stackmap()[%q].Name = %q, want %q",
				stack.Name, gotStack.Name, stack.Name)
		}
	}
}

func TestStackmapEmpty(t *testing.T) {
	got := Stackmap(nil)

	if len(got) != 0 {
		t.Fatalf("Stackmap(nil) returned %d entries, want 0", len(got))
	}
}

func TestMergeMaps(t *testing.T) {
	base := map[string]string{
		"name":    "alice",
		"command": "test",
		"env":     "prod",
	}

	override := map[string]string{
		"name":    "bob",
		"command": "build",
		"unknown": "should-not-be-added",
	}

	got := MergeMaps(base, override)

	if got["name"] != "bob" {
		t.Errorf("name = %q, want bob", got["name"])
	}

	if got["command"] != "build" {
		t.Errorf("command = %q, want build", got["command"])
	}

	if got["env"] != "prod" {
		t.Errorf("env = %q, want prod", got["env"])
	}

	if _, ok := got["unknown"]; ok {
		t.Error("MergeMaps should not add keys that do not exist in base")
	}

	// Make sure the original map wasn't modified.
	if base["name"] != "alice" {
		t.Error("MergeMaps modified the base map")
	}
}

func TestMergeMapsNilBase(t *testing.T) {
	base := map[string]string(nil)

	override := map[string]string{
		"name": "alice",
	}

	got := MergeMaps(base, override)

	if len(got) != 0 {
		t.Fatalf(
			"MergeMaps(nil, override) returned %#v, want empty map",
			got,
		)
	}
}

func TestValidateStackVarsNoRules(t *testing.T) {
	stack := &config.Stack{
		Name: "stack1",
	}

	if err := ValidateStackVars(stack); err != nil {
		t.Fatalf("ValidateStackVars() error = %v, want nil", err)
	}
}

func TestValidateStackVarsNilStack(t *testing.T) {
	if err := ValidateStackVars(nil); err != nil {
		t.Fatalf("ValidateStackVars(nil) error = %v, want nil", err)
	}
}

func TestValidateStackVarsRequired(t *testing.T) {
	stack := &config.Stack{
		Name: "stack1",
		VarRules: map[string]config.VariableRuleSet{
			"environment": {
				Required: true,
			},
		},
		Vars: map[string]string{},
	}

	err := ValidateStackVars(stack)

	if err == nil {
		t.Fatal("ValidateStackVars() error = nil, want required-variable error")
	}

	if !strings.Contains(err.Error(), "environment") {
		t.Fatalf(
			"error = %q, want environment in error",
			err.Error(),
		)
	}
}

func TestValidateStackVarsRequiredEmpty(t *testing.T) {
	stack := &config.Stack{
		Name: "stack1",
		VarRules: map[string]config.VariableRuleSet{
			"environment": {
				Required: true,
			},
		},
		Vars: map[string]string{
			"environment": "   ",
		},
	}

	err := ValidateStackVars(stack)

	if err == nil {
		t.Fatal("ValidateStackVars() error = nil, want required-value error")
	}

	if !strings.Contains(err.Error(), "environment") {
		t.Fatalf(
			"error = %q, want environment in error",
			err.Error(),
		)
	}
}

func TestValidateStackVarsAllowedValues(t *testing.T) {
	stack := &config.Stack{
		Name: "deploy",
		VarRules: map[string]config.VariableRuleSet{
			"environment": {
				AllowedValue: []string{
					"dev",
					"staging",
					"prod",
				},
			},
		},
		Vars: map[string]string{
			"environment": "prod",
		},
	}

	if err := ValidateStackVars(stack); err != nil {
		t.Fatalf("valid value rejected: %v", err)
	}

	stack.Vars["environment"] = "test"

	err := ValidateStackVars(stack)

	if err == nil {
		t.Fatal("invalid allowed value was accepted")
	}

	if !strings.Contains(err.Error(), "must be one of") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateStackVarsDeniedValues(t *testing.T) {
	stack := &config.Stack{
		Name: "deploy",
		VarRules: map[string]config.VariableRuleSet{
			"environment": {
				DeniedValue: []string{"production"},
			},
		},
		Vars: map[string]string{
			"environment": "staging",
		},
	}

	if err := ValidateStackVars(stack); err != nil {
		t.Fatalf("valid value rejected: %v", err)
	}

	stack.Vars["environment"] = "production"

	err := ValidateStackVars(stack)

	if err == nil {
		t.Fatal("denied value was accepted")
	}

	if !strings.Contains(err.Error(), "is denied") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateStackVarsAllowedRegex(t *testing.T) {
	stack := &config.Stack{
		Name: "deploy",
		VarRules: map[string]config.VariableRuleSet{
			"version": {
				AllowedRegex: `regex(^v[0-9]+\.[0-9]+\.[0-9]+$)`,
			},
		},
		Vars: map[string]string{
			"version": "v1.2.3",
		},
	}

	if err := ValidateStackVars(stack); err != nil {
		t.Fatalf("valid regex value rejected: %v", err)
	}

	stack.Vars["version"] = "latest"

	err := ValidateStackVars(stack)

	if err == nil {
		t.Fatal("invalid regex value was accepted")
	}

	if !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateStackVarsDeniedRegex(t *testing.T) {
	stack := &config.Stack{
		Name: "deploy",
		VarRules: map[string]config.VariableRuleSet{
			"branch": {
				DeniedRegex: `regex(^production$)`,
			},
		},
		Vars: map[string]string{
			"branch": "staging",
		},
	}

	if err := ValidateStackVars(stack); err != nil {
		t.Fatalf("valid value rejected: %v", err)
	}

	stack.Vars["branch"] = "production"

	err := ValidateStackVars(stack)

	if err == nil {
		t.Fatal("denied regex value was accepted")
	}

	if !strings.Contains(err.Error(), "matches denied_regex") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateStackVarsInvalidRegexRule(t *testing.T) {
	stack := &config.Stack{
		Name: "stack1",
		VarRules: map[string]config.VariableRuleSet{
			"value": {
				AllowedRegex: "not-a-regex-rule",
			},
		},
		Vars: map[string]string{
			"value": "hello",
		},
	}

	err := ValidateStackVars(stack)

	if err == nil {
		t.Fatal("invalid regex rule was accepted")
	}

	if !strings.Contains(err.Error(), "expected regex(...) format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateStackVarsEmptyRegexRule(t *testing.T) {
	stack := &config.Stack{
		Name: "stack1",
		VarRules: map[string]config.VariableRuleSet{
			"value": {
				AllowedRegex: `regex("")`,
			},
		},
		Vars: map[string]string{
			"value": "hello",
		},
	}

	err := ValidateStackVars(stack)

	if err == nil {
		t.Fatal("empty regex pattern was accepted")
	}

	if !strings.Contains(err.Error(), "empty pattern") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractRegexPattern(t *testing.T) {
	tests := []struct {
		name string
		rule string
		want string
	}{
		{
			name: "plain pattern",
			rule: `regex(^hello$)`,
			want: `^hello$`,
		},
		{
			name: "quoted pattern",
			rule: `regex("^hello$")`,
			want: `^hello$`,
		},
		{
			name: "single quoted pattern",
			rule: `regex('^hello$')`,
			want: `^hello$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractRegexPattern(tt.rule)
			if err != nil {
				t.Fatalf("extractRegexPattern() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractRegexPatternInvalid(t *testing.T) {
	tests := []string{
		"",
		"hello",
		"regex",
		"regex(",
		"regex)",
		`regex("")`,
		`regex('')`,
	}

	for _, rule := range tests {
		t.Run(rule, func(t *testing.T) {
			_, err := extractRegexPattern(rule)
			if err == nil {
				t.Fatalf(
					"extractRegexPattern(%q) error = nil, want error",
					rule,
				)
			}
		})
	}
}

func TestExecuteStackValidationFailure(t *testing.T) {
	stack := &config.Stack{
		Name: "stack1",
		VarRules: map[string]config.VariableRuleSet{
			"required": {
				Required: true,
			},
		},
		Vars: map[string]string{},
	}

	output := "json"

	// Validation happens before shell execution, so an executor with no
	// configuration is sufficient for this test.
	executor := &config.Executor{}

	result := ExecuteStack(executor, stack, &output)

	if result == nil {
		t.Fatal("ExecuteStack() returned nil")
	}

	if result.Success {
		t.Fatal("validation failure returned Success=true")
	}

	if result.Name != "stack1" {
		t.Errorf("Name = %q, want stack1", result.Name)
	}

	if !strings.Contains(result.Error, "required") {
		t.Errorf("Error = %q, want required-variable error", result.Error)
	}

	if result.ContinueOnError {
		t.Error("validation failure should not continue on error")
	}
}

func TestExecuteStackValidationFailureDoesNotExecuteShell(t *testing.T) {
	stack := &config.Stack{
		Name: "stack1",
		VarRules: map[string]config.VariableRuleSet{
			"required": {
				Required: true,
			},
		},
	}

	output := "json"
	executor := &config.Executor{}

	result := ExecuteStack(executor, stack, &output)

	if result == nil {
		t.Fatal("ExecuteStack() returned nil")
	}

	// The validation error proves that execution stopped before
	// shellexecutor.ExecuteShell(), because there is no executable
	// configuration in the test executor.
	if !strings.Contains(result.Error, "variable 'required' is required") {
		t.Fatalf("unexpected error: %q", result.Error)
	}
}
