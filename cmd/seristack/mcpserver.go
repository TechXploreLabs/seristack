package main

import (
	"fmt"
	"slices"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	conf "github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/mcpserver"
	"github.com/TechXploreLabs/seristack/internal/shellexecutor"
)

var (
	mcptype string
	addr    string
)

// runCmd represents the run command
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server, expose stack as tools",
	Long: `
  # Start streamableHTTP
  seristack mcp --type streamableHTTP --port 3000
  
  # Start sse
  seristack mcp --config myconfig.yaml --type sse  --port 9090 --addr 0.0.0.0`,
	RunE: mcpServer,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.Flags().StringVarP(&port, "port", "p", "8080", "mcp server port (overrides config)")
	mcpCmd.Flags().StringVarP(&mcptype, "type", "t", "", "mcp server type sse/streamableHTTP")
	mcpCmd.Flags().StringVarP(&addr, "addr", "a", "127.0.0.1", "addr is 127.0.0.1 or 0.0.0.0")
}

func mcpServer(cmd *cobra.Command, args []string) error {
	mcp_type := []string{"sse", "streamableHTTP"}
	if mcptype != "" && !slices.Contains(mcp_type, mcptype) {
		return fmt.Errorf("%s", color.RedString("Error: supported mcp type sse/streamableHTTP"))
	}
	config, err := conf.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("%s", color.RedString("Error: [failed to load config], %v", err))
	}
	shellexecutor.SetConcurrencyLimit(limit)
	err = mcpserver.McpServer(config, mcptype, port, addr)
	if err != nil {
		return err
	}
	return nil
}
