// Unit tests for LoadConfig in loadconfig.go
package config

import (
	"os"
	"path/filepath"
	"reflect"
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
      key: "value"
    cmds:
      - "echo hello"
`,
			wantConfig: &Config{
				Stacks: []Stack{
					{
						Name:            "stack1",
						WorkDir:         "/tmp",
						ContinueOnError: true,
						Vars:            map[string]string{"key": "value"},
						Cmds:            []string{"echo hello"},
					},
				},
				Server: nil,
			},
			wantErr: false,
		},
		{
			name: "Valid config with stacks and server",
			yamlData: `
stacks:
  - name: "stack2"
    isSerial: true
server:
  host: "localhost"
  port: "8080"
  endpoint:
    - path: "/api"
      method: "GET"
      stackName: "stack2"
`,
			wantConfig: &Config{
				Stacks: []Stack{
					{
						Name:     "stack2",
						IsSerial: true,
					},
				},
				Server: &Serverconfig{
					Host: "localhost",
					Port: "8080",
					Endpoints: []Endpoint{
						{
							Path:      "/api",
							Method:    "GET",
							Stackname: "stack2",
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
