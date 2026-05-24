package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	conf "github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/shellexecutor"
	"github.com/TechXploreLabs/seristack/internal/trigger"
)

var (
	stack    string
	output   string
	vars     []string
	varsJSON string
)

var triggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: "Execute tasks defined in config file",
	Long: `Execute stacks

Examples:
  # Execute all stacks in config.yaml
  seristack trigger

  # Ececute specific stack
  seristack trigger -s stackname 

  # Ececute specific stack with vars
  seristack trigger -s stackname --vars invite=seristack --vars group=automation --vars grade=Lightweight

  # Pass JSON string safely as a single var value
  seristack trigger -s stackname --vars 'payload={"name":"alice","roles":"admin"}'

  # Pass multiple vars in one JSON object
  seristack trigger -s stackname --vars-json '{"name":"alice","env":"dev","payload":{"roles":["admin"]}}'
 `,
	RunE: setupTrigger,
}

func init() {
	rootCmd.AddCommand(triggerCmd)
	triggerCmd.Flags().StringVarP(&stack, "stack", "s", "", "run a particular stack")
	triggerCmd.Flags().StringArrayVarP(&vars, "vars", "v", []string{}, "override variables (key=value)")
	triggerCmd.Flags().StringVar(&varsJSON, "vars-json", "", "override variables as JSON object")
}

func parseVars(varsSlice []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, v := range varsSlice {
		if err := addVarPair(result, v); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func parseVarsJSON(raw string) (map[string]string, error) {
	result := make(map[string]string)
	if strings.TrimSpace(raw) == "" {
		return result, nil
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, fmt.Errorf("invalid --vars-json payload: %w", err)
	}

	for k, v := range m {
		if k == "" {
			return nil, fmt.Errorf("invalid --vars-json payload: empty key")
		}
		switch vv := v.(type) {
		case string:
			result[k] = vv
		default:
			b, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("invalid --vars-json payload for key '%s': %w", k, err)
			}
			result[k] = string(b)
		}
	}

	return result, nil
}

func addVarPair(vars map[string]string, pair string) error {
	parts := strings.SplitN(pair, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format '%s' (expected key=value)", pair)
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" {
		return fmt.Errorf("empty key in '%s'", pair)
	}
	vars[key] = value
	return nil
}

func setupTrigger(cmd *cobra.Command, args []string) error {
	varsMap, err := parseVarsJSON(varsJSON)
	if err != nil {
		return fmt.Errorf("%s", color.RedString("Error: [parsing vars-json], %v", err))
	}

	flagVarsMap, err := parseVars(vars)
	if err != nil {
		return fmt.Errorf("%s", color.RedString("Error: [parsing vars], %v", err))
	}
	for k, v := range flagVarsMap {
		varsMap[k] = v
	}

	config, err := conf.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("%s", color.RedString("Error: [failed to load config], %v", err))
	}
	if stack != "" {
		config, err = trigger.SingleStackCheck(config, &stack)
		if err != nil {
			return fmt.Errorf("%s", color.RedString("%v", err))
		}
	}
	shellexecutor.SetConcurrencyLimit(limit)
	trigger.RunTrigger(config, &output, &varsMap)
	return nil
}
