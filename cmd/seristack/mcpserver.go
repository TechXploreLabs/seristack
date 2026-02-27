package main

import (
	"fmt"
	"slices"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	conf "github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/mcpserver"
)

var (
	mcptype string
	addr    string
)

// runCmd represents the run command
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start HTTP server to expose tasks as API endpoints",
	Long: `Run starts an HTTP server that exposes your tasks as REST API endpoints.
	
Examples:
  # Start stdio server with default config
  seristack mcp
  
  # Start streamableHTTP
  seristack mcp --type streamableHTTP --port 3000
  
  # Start sse
  seristack run --config myconfig.yaml --type sse  --port 9090`,
	RunE: mcpServer,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.Flags().StringVarP(&port, "port", "p", "", "mcp server port (overrides config)")
	mcpCmd.Flags().StringVarP(&mcptype, "type", "t", "", "mcp server type stdio/sse/streamableHTTP")
	mcpCmd.Flags().StringVarP(&addr, "addr", "a", "", "addr is 127.0.0.1,0.0.0.0")
}

func mcpServer(cmd *cobra.Command, args []string) error {
	mcp_type := []string{"sse", "stdio", "streamableHTTP"}
	if mcptype != "" && !slices.Contains(mcp_type, mcptype) {
		return fmt.Errorf("%s", color.RedString("Error: supported mcp type stdio/sse/streamableHTTP"))
	}
	config, err := conf.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("%s", color.RedString("Error: [failed to load config], %v", err))
	}
	mcpserver.McpServer(config, mcptype, port, addr)
	return nil
}
