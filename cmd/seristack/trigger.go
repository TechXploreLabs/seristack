package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	conf "github.com/TechXploreLabs/seristack/internal/config"
	exe "github.com/TechXploreLabs/seristack/internal/executehandler"
	"github.com/TechXploreLabs/seristack/internal/function"
	"github.com/TechXploreLabs/seristack/internal/registry"
)

var (
	stackName string
	dryRun    bool
)

var triggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: "Execute tasks defined in config file",
	Long: `Execute tasks with dependency resolution using Kahn's algorithm.
	
Examples:
  # Execute all stacks in config.yaml
  seristack trigger
  
  # Execute with custom config file
  seristack trigger --config myconfig.yaml
  
  # Dry run (show execution order without running)
  seristack trigger --dry-run`,
	RunE: runTrigger,
}

func init() {
	rootCmd.AddCommand(triggerCmd)
	triggerCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show execution plan without running")
}

func runTrigger(cmd *cobra.Command, args []string) error {
	config, err := conf.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if verbose {
		fmt.Printf("Loaded config from: %s\n", configFile)
		fmt.Printf("Found %d stacks\n", len(config.Stacks))
	}
	order, err := function.ExecutionOrder(config.Stacks)
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %w", err)
	}
	if dryRun {
		fmt.Println("Execution Plan:")
		for i, batch := range order {
			fmt.Printf("\nBatch %d (parallel):\n", i+1)
			for _, stackName := range batch {
				fmt.Printf("  - %s\n", stackName)
			}
		}
		return nil
	}
	sourceDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	executor := &conf.Executor{
		Registry:  registry.NewRegistry(&order),
		Config:    config,
		SourceDir: sourceDir,
	}

	if verbose {
		fmt.Println("Starting execution...")
	}
	if err := exe.Execute(executor, &order); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}
	if verbose {
		fmt.Println("✓ Execution completed successfully")
	}
	return nil
}
