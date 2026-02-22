package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	Execute()
}

var (
	configFile string
)

var rootCmd = &cobra.Command{
	Use:           "seristack",
	Short:         "Seristack - A modern task automation tool",
	SilenceErrors: true,
	SilenceUsage:  true,
	Long: `Seristack is a task automation tool that allows you to:
- Execute tasks with dependency management (trigger)
- Run tasks as HTTP API endpoints (run)

Visit https://github.com/TechXploreLabs/seristack for more information.`,
	Version: "0.0.5",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yaml", "config file")
}
