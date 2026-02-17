package main

import (
	"os"

	"github.com/fatih/color"
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
		color.Red("Error: failed to load config: %v", err)
		os.Exit(1)
	}

	if config.Server == nil {
		color.Red("Error: server configuration is missing")
		os.Exit(1)
	}

	if port != "" {
		config.Server.Port = port
	}

	server.Server(config)
	return nil
}
