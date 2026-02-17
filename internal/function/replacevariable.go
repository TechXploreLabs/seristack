package function

import (
	"bytes"
	"text/template"

	"github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/registry"
)

func ReplaceVariables(vars map[string]string, result *config.Executor, input string) (string, error) {
	if result == nil {
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

	data := config.VariableSubstitution{
		Vars:   vars,
		Result: registry.GetAllVars(result.Registry),
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
