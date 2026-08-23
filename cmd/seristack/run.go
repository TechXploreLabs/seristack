package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/TechXploreLabs/seristack/internal/audit"
	conf "github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/server"
	"github.com/TechXploreLabs/seristack/internal/shellexecutor"
)

var (
	port         string
	auditLogPath string
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start HTTP server to expose tasks as API endpoints",
	Long: `Run starts an HTTP server that exposes your tasks as REST API endpoints.
 
Production note:
  Seristack can execute shell commands. For public or shared environments,
  bind Seristack to 127.0.0.1 or a private network and put Nginx/Caddy in
  front for TLS, authentication.
	
Examples:
  # Start server with default config
  seristack run
 
  # Start with custom port
  seristack run --port 3000
 
  # Start with custom config and host
  seristack run --config myconfig.yaml --port 9090 --addr 127.0.0.1
 
  # Start with audit log enabled
  seristack run --audit-log /var/log/seristack/audit.log`,
	RunE: runServer,
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&port, "port", "p", "8080", "server port")
	runCmd.Flags().StringVarP(&addr, "addr", "a", "127.0.0.1", "bind address (127.0.0.1 or 0.0.0.0)")
	runCmd.Flags().StringVar(&auditLogPath, "audit-log", "", "path to audit log file (enables audit logging when set)")
}

func runServer(cmd *cobra.Command, args []string) error {
	config, err := conf.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("%s", color.RedString("Error: [failed to load config], %v", err))
	}
	shellexecutor.SetConcurrencyLimit(limit)
	var auditLogger *audit.Logger
	if auditLogPath != "" {
		auditLogger, err = audit.New(auditLogPath)
		if err != nil {
			return fmt.Errorf("%s", color.RedString("Error: [failed to initialise audit log], %v", err))
		}
		defer auditLogger.Close()
	}
	return server.Server(config, &port, &addr, auditLogger)
}
