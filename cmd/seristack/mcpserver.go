package main

import (
	"fmt"
	"slices"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/TechXploreLabs/seristack/internal/audit"
	conf "github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/mcpserver"
	"github.com/TechXploreLabs/seristack/internal/shellexecutor"
)

var (
	mcptype                string
	addr                   string
	mcpAuditLogPath        string
	excludeIdentityHeaders []string
)

// runCmd represents the run command
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server, expose stack as tools",
	Long: `
  Production note:
    MCP tools can trigger configured Seristack stacks. For public or shared
    environments, bind to 127.0.0.1 or a private network and expose through
    Nginx/Caddy for TLS, authentication.

  # Start streamableHTTP
  seristack mcp --type streamableHTTP --port 3000
  
  # Start sse
  seristack mcp --config myconfig.yaml --type sse  --port 9090 --addr 0.0.0.0

  # streamableHTTP with audit log
    seristack mcp --type streamableHTTP --port 8081 \
      --audit-log /var/log/seristack/mcp-audit.log

  # exclude specific headers from audit log
	seristack mcp --type streamableHTTP --port 8081 \
		--audit-log /var/log/seristack/mcp-audit.log \
		--exclude-identity-headers "X-Internal-Token" \
		--exclude-identity-headers "X-Debug-Header"`,
	RunE: mcpServer,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.Flags().StringVarP(&port, "port", "p", "8080", "mcp server port (overrides config)")
	mcpCmd.Flags().StringVarP(&mcptype, "type", "t", "stdio", "mcp server type stdio/sse/streamableHTTP")
	mcpCmd.Flags().StringVarP(&addr, "addr", "a", "127.0.0.1", "addr is 127.0.0.1 or 0.0.0.0")
	mcpCmd.Flags().StringVar(&mcpAuditLogPath, "audit-log", "", "path to audit log file (enables audit logging when set)")
	mcpCmd.Flags().StringArrayVar(&excludeIdentityHeaders, "exclude-identity-headers", nil, "exclude headers appearing in the audit log")
}

func mcpServer(cmd *cobra.Command, args []string) error {
	mcp_type := []string{"stdio", "sse", "streamableHTTP"}
	if mcptype != "" && !slices.Contains(mcp_type, mcptype) {
		return fmt.Errorf("%s", color.RedString("Error: supported mcp type sse/streamableHTTP"))
	}
	config, err := conf.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("%s", color.RedString("Error: [failed to load config], %v", err))
	}
	shellexecutor.SetConcurrencyLimit(limit)
	var auditLogger *audit.Logger
	if mcpAuditLogPath != "" {
		auditLogger, err = audit.New(mcpAuditLogPath)
		if err != nil {
			return fmt.Errorf("%s", color.RedString("Error: [failed to initialise audit log], %v", err))
		}
		defer auditLogger.Close()
	}
	err = mcpserver.McpServer(config, mcptype, port, addr, auditLogger, excludeIdentityHeaders)
	if err != nil {
		return err
	}
	return nil
}
