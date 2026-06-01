package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load the yaml file for config extraction

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	for i := range config.Stacks {
		if err := NormalizeStackVariables(&config.Stacks[i]); err != nil {
			return nil, err
		}
	}

	return &config, nil
}

func (v VariableRuleSet) IsEmpty() bool {
	return len(v.AllowedValue) == 0 && len(v.DeniedValue) == 0 && strings.TrimSpace(v.AllowedRegex) == "" && strings.TrimSpace(v.DeniedRegex) == "" && !v.Required
}

func NormalizeStackVariables(stack *Stack) error {
	stack.Vars = make(map[string]string)
	stack.VarRules = make(map[string]VariableRuleSet)

	for idx, variable := range stack.Variables {
		name := strings.TrimSpace(variable.Name)
		if name == "" {
			return fmt.Errorf("invalid vars[%d]: name is required", idx)
		}
		if _, exists := stack.Vars[name]; exists {
			return fmt.Errorf("invalid vars[%d]: duplicate variable name '%s'", idx, name)
		}

		stack.Vars[name] = variable.Value
		rule := VariableRuleSet{
			AllowedValue: variable.AllowedValue,
			DeniedValue:  variable.DeniedValue,
			AllowedRegex: strings.TrimSpace(variable.AllowedRegex),
			DeniedRegex:  strings.TrimSpace(variable.DeniedRegex),
			Required:     variable.Required,
		}

		ruleSetCount := 0
		if len(rule.AllowedValue) > 0 {
			ruleSetCount++
		}
		if len(rule.DeniedValue) > 0 {
			ruleSetCount++
		}
		if rule.AllowedRegex != "" {
			ruleSetCount++
		}
		if rule.DeniedRegex != "" {
			ruleSetCount++
		}
		if ruleSetCount > 1 {
			return fmt.Errorf("invalid vars[%d] (%s): only one ruleset is allowed among allowed_value, denied_value, allowed_regex, denied_regex", idx, name)
		}

		if rule.AllowedRegex != "" && (!strings.HasPrefix(rule.AllowedRegex, "regex(") || !strings.HasSuffix(rule.AllowedRegex, ")")) {
			return fmt.Errorf("invalid vars[%d].allowed_regex: expected regex(...) format", idx)
		}
		if rule.DeniedRegex != "" && (!strings.HasPrefix(rule.DeniedRegex, "regex(") || !strings.HasSuffix(rule.DeniedRegex, ")")) {
			return fmt.Errorf("invalid vars[%d].denied_regex: expected regex(...) format", idx)
		}

		if !rule.IsEmpty() {
			stack.VarRules[name] = rule
		}
	}

	if stack.Vars == nil {
		stack.Vars = map[string]string{}
	}
	if stack.VarRules == nil {
		stack.VarRules = map[string]VariableRuleSet{}
	}

	return nil
}
