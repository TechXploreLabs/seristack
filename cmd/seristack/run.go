package main

import (
	"fmt"

	"github.com/spf13/cobra"

	conf "github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/server"
)

var (
	port string
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start HTTP server to expose tasks as API endpoints",
	Long: `Run starts an HTTP server that exposes your tasks as REST API endpoints.
	
Examples:
  # Start server with default config
  seristack run
  
  # Start with custom port
  seristack run --port 3000
  
  # Start with custom config and host
  seristack run --config myconfig.yaml --port 9090`,
	RunE: runServer,
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Run-specific flags
	runCmd.Flags().StringVarP(&port, "port", "p", "", "server port (overrides config)")
}

func runServer(cmd *cobra.Command, args []string) error {
	config, err := conf.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if config.Server == nil {
		return fmt.Errorf("server configuration is missing")
	}

	if port != "" {
		config.Server.Port = port
	}

	if verbose {
		fmt.Printf("Loaded config from: %s\n", configFile)
		fmt.Printf("Registered %d endpoints\n", len(config.Server.Endpoints))
	}
	server.Server(config)
	return nil
}
