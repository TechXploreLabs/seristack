package function

import (
	"bytes"
	"regexp"
	"text/template"

	"github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/registry"
)

var (
	resultDotPattern = regexp.MustCompile(`\{\{\s*\.Result\.([A-Za-z0-9_-]+)`)
)

func ReplaceVariables(vars map[string]string, result *config.Executor, input string) (string, error) {
	if result == nil || result.Registry == nil {
		data := map[string]any{"Vars": vars}
		tmpl, err := template.New("shellScript1").Parse(input)
		if err != nil {
			return "Error: Variable Template", err
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return "Error: Variable Substitution", err
		}
		modifiedCmd := buf.String()
		return modifiedCmd, nil
	}

	resultKeys := collectResultKeys(input)
	resultVars := map[string]string{}
	if len(resultKeys) > 0 {
		resultVars = registry.GetVarsByNames(result.Registry, resultKeys)
	}

	data := config.VariableSubstitution{
		Vars:   vars,
		Result: resultVars,
	}
	tmpl, err := template.New("shellScript1").Parse(input)
	if err != nil {
		return "Error: Variable Template", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "Error: Variable Substitution", err
	}
	modifiedCmd := buf.String()
	return modifiedCmd, nil

}

func collectResultKeys(input string) []string {
	if input == "" {
		return nil
	}

	seen := make(map[string]struct{})
	for _, match := range resultDotPattern.FindAllStringSubmatch(input, -1) {
		if len(match) > 1 && match[1] != "" {
			seen[match[1]] = struct{}{}
		}
	}

	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}

	return keys
}
