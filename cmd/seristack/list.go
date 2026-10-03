package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	conf "github.com/TechXploreLabs/seristack/internal/config"
)

var listCmd = &cobra.Command{
	Use: "list",
	Short: "List the stack",
	Long: `
  To list the stack exist in the --config file
  
  seristack list -c config.yaml`,
	RunE: list,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func list(cmd *cobra.Command, args []string) error {
	config, err := conf.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("%s", color.RedString("Error: [failed to load config], %v", err))
	}
	for _, stacklist := range config.Stacks {
		fmt.Println(stacklist.Name)
	}
	return nil
}