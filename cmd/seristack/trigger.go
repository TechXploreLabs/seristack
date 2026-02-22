package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"slices"

	conf "github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/trigger"
)

var (
	stack  string
	output string
	vars   []string
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
  seristack trigger -s stackname --vars invite=seristack --vars "group=automation,grage=Lightweight"
 `,
	RunE: setupTrigger,
}

func init() {
	rootCmd.AddCommand(triggerCmd)
	triggerCmd.Flags().StringVarP(&stack, "stack", "s", "", "run a particular stack")
	triggerCmd.Flags().StringVarP(&output, "output", "o", "", "output format for result yaml or json")
	triggerCmd.Flags().StringSliceVarP(&vars, "vars", "v", []string{}, "override variables (key=value)")
}

func parseVars(varsSlice []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, v := range varsSlice {
		if strings.Contains(v, ",") {
			pairs := strings.SplitSeq(v, ",")
			for pair := range pairs {
				if err := addVarPair(result, strings.TrimSpace(pair)); err != nil {
					return nil, err
				}
			}
		} else {
			if err := addVarPair(result, v); err != nil {
				return nil, err
			}
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
	varsMap, err := parseVars(vars)
	if err != nil {
		return fmt.Errorf("%s", color.RedString("Error: [parsing vars], %v", err))
	}
	outputformat := []string{"yaml", "json"}
	if output != "" && !slices.Contains(outputformat, output) {
		return fmt.Errorf("%s", color.RedString("Error: supported output format is yaml/json"))
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
	consolidatedresult := trigger.RunTrigger(config, &output, &varsMap)
	if consolidatedresult != nil {
		if output == "yaml" {
			yamldata, _ := yaml.Marshal(map[string]any{"result": &consolidatedresult})
			fmt.Println(string(yamldata))
		} else {
			jsondata, _ := json.MarshalIndent(map[string]any{"result": &consolidatedresult}, "", " ")
			fmt.Println(string(jsondata))
		}
	}
	return nil
}
