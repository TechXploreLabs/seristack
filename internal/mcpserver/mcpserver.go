package mcpserver

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	conf "github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/executehandler"
	"gopkg.in/yaml.v3"
)

func McpServer(config *conf.Config, transport string, port string, addr string) error {
	s := server.NewMCPServer(
		"seristack",
		"0.0.6",
		server.WithToolCapabilities(true),
	)
	stackMap := executehandler.Stackmap(config.Stacks)
	for _, stack := range config.Stacks {
		if stack.Description != "" {
			registerStackTool(s, stack, stackMap)
		}
	}
	switch transport {
	case "sse":
		if port == "" {
			port = "8080"
		}
		if addr == "" {
			addr = "127.0.0.1"
		}
		sseServer := server.NewSSEServer(s, server.WithBaseURL("http://"+addr+":"+port))
		fmt.Printf("MCP SSE server starting on http://%s:%s/sse\n", addr, port)
		return sseServer.Start(":" + port)
	case "streamableHTTP":
		if port == "" {
			port = "8080"
		}
		httpServer := server.NewStreamableHTTPServer(s)
		fmt.Printf("MCP Streamable HTTP server starting on http://127.0.0.1:%s/mcp\n", port)
		return httpServer.Start(":" + port)
	default:
		return fmt.Errorf("streamableHTTP or sse")
	}
}

func registerStackTool(s *server.MCPServer, stack conf.Stack, stackMap map[string]*conf.Stack) {
	options := []mcp.ToolOption{
		mcp.WithDescription(stack.Description),
	}
	for varName := range stack.Vars {
		options = append(options, mcp.WithString(varName,
			mcp.Description(fmt.Sprintf("Variable '%s'for stack '%s'", varName, stack.Name)),
		))
	}
	tool := mcp.NewTool(stack.Name, options...)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Printf("Tool called: tool: %s, args: %s", stack.Name, req.Params.Arguments)
		output := "yaml"
		sourceDir, _ := os.Getwd()
		vars := make(map[string]string)
		if args, ok := req.Params.Arguments.(map[string]interface{}); ok {
			for k, v := range args {
				vars[k] = fmt.Sprintf("%v", v)
			}
		}
		executor := &conf.Executor{
			Registry:  nil,
			Config:    nil,
			SourceDir: sourceDir,
		}
		stackMap[stack.Name].Vars = vars
		result := executehandler.ExecuteStack(executor, stackMap[stack.Name], &output)
		yamldata, _ := yaml.Marshal(result)
		log.Printf("Tool execution completed: tool: %s", stack.Name)
		return mcp.NewToolResultText(string(yamldata)), nil
	})
}
