// Unit tests for LoadConfig in loadconfig.go
package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpfile := filepath.Join(os.TempDir(), "tmpconfig.yaml")
	err := os.WriteFile(tmpfile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	return tmpfile
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name       string
		yamlData   string
		wantConfig *Config
		wantErr    bool
	}{
		{
			name: "Valid config with stacks only",
			yamlData: `
stacks:
  - name: "stack1"
    workDir: "/tmp"
    continueOnError: true
    vars:
      - name: "key"
        value: "value"
        allowed_value: ["value", "devvalue"]
    cmds:
      - "echo hello"
`,
			wantConfig: &Config{
				Stacks: []Stack{
					{
						Name:            "stack1",
						WorkDir:         "/tmp",
						ContinueOnError: true,
						Variables: []VariableDef{
							{
								Name:         "key",
								Value:        "value",
								AllowedValue: []string{"value", "devvalue"},
							},
						},
						Vars:     map[string]string{"key": "value"},
						VarRules: map[string]VariableRuleSet{"key": {AllowedValue: []string{"value", "devvalue"}}},
						Cmds:     []string{"echo hello"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid config with stacks and server",
			yamlData: `
stacks:
  - name: "stack2"
    executionMode: PARALLEL
`,
			wantConfig: &Config{
				Stacks: []Stack{
					{
						Name:          "stack2",
						ExecutionMode: "PARALLEL",
						Vars:          map[string]string{},
						VarRules:      map[string]VariableRuleSet{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid old vars map format",
			yamlData: `
stacks:
  - name: "stack3"
    vars:
      key: "value"
`,
			wantErr: true,
		},
		{
			name: "Invalid conflicting variable rules",
			yamlData: `
stacks:
  - name: "stack4"
    vars:
      - name: "invite"
        value: "hello"
        allowed_value: ["hello"]
        denied_regex: regex("^h.*")
`,
			wantErr: true,
		},
		{
			name: "Required-only rule is preserved in VarRules",
			yamlData: `
stacks:
  - name: "stack_required"
    vars:
      - name: "token"
        value: ""
        required: true
`,
			wantConfig: &Config{
				Stacks: []Stack{
					{
						Name: "stack_required",
						Variables: []VariableDef{
							{
								Name:     "token",
								Value:    "",
								Required: true,
							},
						},
						Vars: map[string]string{"token": ""},
						VarRules: map[string]VariableRuleSet{
							"token": {Required: true},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "Invalid YAML syntax",
			yamlData: ": bad yaml", // truly invalid YAML, should cause error
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile := writeTempFile(t, tt.yamlData)
			defer os.Remove(tmpfile)

			got, err := LoadConfig(tmpfile)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got error: %v", tt.wantErr, err)
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.wantConfig) {
					t.Errorf("Loaded config does not match expected.\nGot: %+v\nWant: %+v", got, tt.wantConfig)
				}
			} else if tt.name == "Invalid conflicting variable rules" {
				if err == nil || !strings.Contains(err.Error(), "only one ruleset is allowed") {
					t.Fatalf("expected conflict ruleset error, got: %v", err)
				}
			}
		})
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/path/to/nonexistent.yaml")
	if err == nil {
		t.Errorf("expected error for missing file, got nil")
	}
}
