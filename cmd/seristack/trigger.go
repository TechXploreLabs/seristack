package main

import (
	"fmt"
	"os"

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
)

var triggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: "Execute tasks defined in config file",
	Long: `Execute stacks

Examples:
  # Execute all stacks in config.yaml
  seristack trigger
  
  # Execute with custom config file
  seristack trigger --config myconfig.yaml
  
  # Execute particular stack
  seristack trigger --stack stackname
  
  # Output in yaml format
  seristack trigger -o yaml `,
	RunE: setupTrigger,
}

func init() {
	rootCmd.AddCommand(triggerCmd)
	triggerCmd.Flags().StringVarP(&stack, "stack", "s", "", "run a particular stack")
	triggerCmd.Flags().StringVarP(&output, "output", "o", "", "output format for result")
}

func setupTrigger(cmd *cobra.Command, args []string) error {
	outputformat := []string{"yaml"}
	if output != "" && !slices.Contains(outputformat, output) {
		color.Red("supported output format is yaml")
		os.Exit(1)
	}
	config, err := conf.LoadConfig(configFile)
	if err != nil {
		color.Red("failed to load config: %v", err)
		os.Exit(1)
	}
	if stack != "" {
		config = trigger.SingleStackCheck(config, &stack)
	}
	consolidatedresult := trigger.RunTrigger(config, &output)
	if consolidatedresult != nil {
		yamldata, _ := yaml.Marshal(&consolidatedresult)
		fmt.Printf("result:\n")
		fmt.Println(string(yamldata))
	}
	return nil
}
