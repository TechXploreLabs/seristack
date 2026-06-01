// Unit tests for executehandler.go
package executehandler

import (
	"reflect"
	"strings"
	"testing"

	"github.com/TechXploreLabs/seristack/internal/config"
)

func TestStackmap(t *testing.T) {
	stacks := []config.Stack{
		{Name: "stack1"},
		{Name: "stack2"},
	}

	got := Stackmap(stacks)
	if len(got) != 2 {
		t.Errorf("expected map of 2, got %d", len(got))
	}
	if got["stack1"] == nil || got["stack1"].Name != "stack1" {
		t.Errorf("missing or incorrect entry for 'stack1'")
	}
	if got["stack2"] == nil || got["stack2"].Name != "stack2" {
		t.Errorf("missing or incorrect entry for 'stack2'")
	}

	// Edge case: empty input
	stacks = []config.Stack{}
	got = Stackmap(stacks)
	if !reflect.DeepEqual(got, map[string]*config.Stack{}) {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestValidateStackVars(t *testing.T) {
	tests := []struct {
		name        string
		stack       *config.Stack
		expectError bool
		errContains string
	}{
		{
			name: "regex validation success",
			stack: &config.Stack{
				Variables: []config.VariableDef{
					{
						Name:         "invite",
						Value:        "hello engineers",
						AllowedRegex: "regex(\"^[a-z]{5,}.*$\")",
					},
				},
				Vars: map[string]string{
					"invite": "hello engineers",
				},
				VarRules: map[string]config.VariableRuleSet{
					"invite": {AllowedRegex: "regex(\"^[a-z]{5,}.*$\")"},
				},
			},
			expectError: false,
		},
		{
			name: "enum validation success",
			stack: &config.Stack{
				Variables: []config.VariableDef{
					{
						Name:         "samplekey",
						Value:        "samplevalue",
						AllowedValue: []string{"samplevalue", "devvalue"},
					},
				},
				Vars: map[string]string{
					"samplekey": "samplevalue",
				},
				VarRules: map[string]config.VariableRuleSet{
					"samplekey": {AllowedValue: []string{"samplevalue", "devvalue"}},
				},
			},
			expectError: false,
		},
		{
			name: "regex validation failure",
			stack: &config.Stack{
				Variables: []config.VariableDef{
					{
						Name:         "invite",
						Value:        "HELLO",
						AllowedRegex: "regex(\"^[a-z]{5,}.*$\")",
					},
				},
				Vars: map[string]string{
					"invite": "HELLO",
				},
				VarRules: map[string]config.VariableRuleSet{
					"invite": {AllowedRegex: "regex(\"^[a-z]{5,}.*$\")"},
				},
			},
			expectError: true,
			errContains: "does not match allowed_regex",
		},
		{
			name: "enum validation failure",
			stack: &config.Stack{
				Variables: []config.VariableDef{
					{
						Name:         "samplekey",
						Value:        "prodvalue",
						AllowedValue: []string{"samplevalue", "devvalue"},
					},
				},
				Vars: map[string]string{
					"samplekey": "prodvalue",
				},
				VarRules: map[string]config.VariableRuleSet{
					"samplekey": {AllowedValue: []string{"samplevalue", "devvalue"}},
				},
			},
			expectError: true,
			errContains: "must be one of",
		},
		{
			name: "denied value validation failure",
			stack: &config.Stack{
				Variables: []config.VariableDef{{Name: "samplekey", Value: "prodvalue", DeniedValue: []string{"prodvalue"}}},
				Vars:      map[string]string{"samplekey": "prodvalue"},
				VarRules:  map[string]config.VariableRuleSet{"samplekey": {DeniedValue: []string{"prodvalue"}}},
			},
			expectError: true,
			errContains: "is denied",
		},
		{
			name: "denied regex validation failure",
			stack: &config.Stack{
				Variables: []config.VariableDef{{Name: "invite", Value: "admin-user", DeniedRegex: "regex(\"^admin.*$\")"}},
				Vars:      map[string]string{"invite": "admin-user"},
				VarRules:  map[string]config.VariableRuleSet{"invite": {DeniedRegex: "regex(\"^admin.*$\")"}},
			},
			expectError: true,
			errContains: "matches denied_regex",
		},
		{
			name: "required variable validation failure when value is blank",
			stack: &config.Stack{
				Variables: []config.VariableDef{{Name: "token", Value: "   ", Required: true}},
				Vars:      map[string]string{"token": "   "},
				VarRules:  map[string]config.VariableRuleSet{"token": {Required: true}},
			},
			expectError: true,
			errContains: "value for variable 'token' is required",
		},
		{
			name: "required variable validation success when value is present",
			stack: &config.Stack{
				Variables: []config.VariableDef{{Name: "token", Value: "abc", Required: true}},
				Vars:      map[string]string{"token": "abc"},
				VarRules:  map[string]config.VariableRuleSet{"token": {Required: true}},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStackVars(tt.stack)
			if tt.expectError && err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.expectError && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("expected error to contain %q, got %q", tt.errContains, err.Error())
			}
		})
	}
}
