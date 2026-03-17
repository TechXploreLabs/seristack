package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	conf "github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/server"
)

var (
	port     string
	skiproot bool
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
  seristack run --config myconfig.yaml --port 9090

  # Start with custom config and host
  seristack run --config myconfig.yaml --port 9090 --addr 127.0.0.1 --skip-root`,
	RunE: runServer,
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&port, "port", "p", "8080", "server port (overrides config)")
	runCmd.Flags().StringVarP(&addr, "addr", "a", "127.0.0.1", "addr is 127.0.0.1 or 0.0.0.0")
	runCmd.Flags().BoolVarP(&skiproot, "skip-root", "", false, "Skip root disable root trigger")
}

func runServer(cmd *cobra.Command, args []string) error {
	config, err := conf.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("%s", color.RedString("Error: [failed to load config], %v", err))
	}
	err = server.Server(config, &port, &addr, &skiproot)
	if err != nil {
		return err
	}
	return nil
}
