package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/TechXploreLabs/seristack/internal/audit"
	conf "github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/executehandler"
)

type contextKey string

const mcpIdentityKey contextKey = "seristack.mcp.identity"

func McpServer(config *conf.Config, transport string, port string, addr string, auditLogger *audit.Logger, excludeidentityheaders []string) error {
	sourceDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	stackMap := executehandler.Stackmap(config.Stacks)
	s := server.NewMCPServer(
		"seristack",
		"0.4.7",
		server.WithToolCapabilities(true),
		server.WithToolFilter(mcpToolFilter(stackMap)),
	)
	hasRoutes := false
	var registeredPatterns = make(map[string]bool)
	for _, stack := range config.Stacks {
		if stack.Description != "" {
			if registeredPatterns[stack.Name] {
				return fmt.Errorf("duplicate tool registration:  %q ", stack.Name)
			}
			registerStackTool(s, stack, stackMap, sourceDir, auditLogger, excludeidentityheaders)
			hasRoutes = true
			registeredPatterns[stack.Name] = true
		}
	}
	if !hasRoutes {
		return fmt.Errorf("no tools to add — set description on stacks to expose them as MCP tools")
	}
	switch transport {
	case "stdio":
		fmt.Fprintf(os.Stderr, "MCP stdio server starting\n")
		return server.ServeStdio(s)
	case "sse":
		sseServer := server.NewSSEServer(s, server.WithBaseURL("http://"+addr+":"+port))
		handler := mcpIdentityMiddleware(sseServer)
		fmt.Printf("MCP SSE server starting on http://%s:%s/sse\n", addr, port)
		srv := &http.Server{
			Addr:              addr + ":" + port,
			Handler:           handler,
			ReadHeaderTimeout: 30 * time.Second,
			IdleTimeout:       60 * time.Second,
		}
		return srv.ListenAndServe()

	case "streamableHTTP":
		httpServer := server.NewStreamableHTTPServer(s,
			server.WithSessionIdleTTL(30*time.Minute),
			server.WithDisableLocalhostProtection(true),
		)
		handler := mcpIdentityMiddleware(httpServer)
		fmt.Printf("MCP Streamable HTTP server starting on http://%s:%s/mcp\n", addr, port)
		srv := &http.Server{
			Addr:              addr + ":" + port,
			Handler:           handler,
			ReadHeaderTimeout: 30 * time.Second,
			IdleTimeout:       60 * time.Second,
		}
		return srv.ListenAndServe()

	default:
		return fmt.Errorf("unsupported transport %q — use stdio, sse, or streamableHTTP", transport)
	}
}

func mcpToolFilter(
	stackMap map[string]*conf.Stack,
) server.ToolFilterFunc {
	return func(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
		filtered := make([]mcp.Tool, 0, len(tools))

		for _, tool := range tools {
			stack, ok := stackMap[tool.Name]
			if !ok {
				continue
			}

			if checkMCPAccess(ctx, stack.Access, stack.MatchAccess) {
				filtered = append(filtered, tool)
			}
		}

		return filtered
	}
}

func mcpIdentityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity := make(map[string][]string)
		for key, values := range r.Header {
			if len(values) > 0 {
				identity[key] = values
			}
		}
		ctx := context.WithValue(r.Context(), mcpIdentityKey, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getHeaderValues(
	identity map[string][]string,
	headerName string,
) []string {
	var result []string

	for _, value := range identity[headerName] {
		for _, item := range strings.Split(value, ",") {
			item = strings.TrimSpace(item)

			if item != "" {
				result = append(result, item)
			}
		}
	}

	return result
}

func checkMCPAccess(ctx context.Context, rules []conf.AccessRule, matchAccess string) bool {
	if len(rules) == 0 {
		return true
	}

	identity, ok := ctx.Value(mcpIdentityKey).(map[string][]string)
	if !ok || len(identity) == 0 {
		return false
	}

	if strings.ToUpper(matchAccess) == "ALL" {
		for _, rule := range rules {
			userValues := getHeaderValues(identity, rule.HeaderName)
			matched := false
			for _, allowed := range rule.HeaderValue {
				if slices.Contains(userValues, allowed) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
		return true
	}

	for _, rule := range rules {
		userValues := getHeaderValues(identity, rule.HeaderName)
		for _, allowed := range rule.HeaderValue {
			if slices.Contains(userValues, allowed) {
				return true
			}
		}
	}
	return false
}

func identityFromContext(ctx context.Context) map[string][]string {
	identity, ok := ctx.Value(mcpIdentityKey).(map[string][]string)
	if !ok || len(identity) == 0 {
		return nil
	}
	return identity
}

func flattenIdentity(identity map[string][]string, excludeidentityheaders []string) map[string]string {
	if len(identity) == 0 {
		return nil
	}
	result := make(map[string]string)
	for k, v := range identity {
		if slices.Contains(excludeidentityheaders, k) {
			continue
		}
		result[k] = strings.Join(v, ",")
	}
	return result
}

func registerStackTool(s *server.MCPServer, stack conf.Stack, stackMap map[string]*conf.Stack, sourceDir string, auditLogger *audit.Logger, excludeidentityheaders []string) {
	options := []mcp.ToolOption{
		mcp.WithDescription(stack.Description),
	}
	for _, varName := range stack.Variables {
		desc := fmt.Sprintf("Variable '%s' for stack '%s'.", varName.Name, stack.Name)

		if len(varName.AllowedValue) > 0 {
			desc += fmt.Sprintf(" Allowed values: %s.", strings.Join(varName.AllowedValue, ", "))
		}
		if len(varName.DeniedValue) > 0 {
			desc += fmt.Sprintf(" Denied values: %s.", strings.Join(varName.DeniedValue, ", "))
		}
		if varName.AllowedRegex != "" {
			desc += fmt.Sprintf(" Must match regex: %s.", varName.AllowedRegex)
		}
		if varName.DeniedRegex != "" {
			desc += fmt.Sprintf(" Must NOT match regex: %s.", varName.DeniedRegex)
		}
		stringOpts := []mcp.PropertyOption{
			mcp.Description(desc),
		}
		if varName.Required {
			stringOpts = append(stringOpts, mcp.Required())
		}
		options = append(options, mcp.WithString(varName.Name, stringOpts...))
	}
	tool := mcp.NewTool(stack.Name, options...)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		vars := make(map[string]string)
		log.Printf("Tool called: tool: %s, args: %v", stack.Name, req.Params.Arguments)
		if !checkMCPAccess(ctx, stack.Access, stack.MatchAccess) {
			if auditLogger != nil {
				if err := auditLogger.Write(audit.Entry{
					Stack:      stack.Name,
					Identity:   flattenIdentity(identityFromContext(ctx), excludeidentityheaders),
					Success:    false,
					DurationMs: time.Since(start).Milliseconds(),
					Error:      "access denied: insufficient permissions to call this tool",
				}); err != nil {
					log.Printf("audit log write failed: %v", err)
				}
			}
			log.Printf("MCP access denied: tool: %s", stack.Name)
			return mcp.NewToolResultError("access denied: insufficient permissions to call this tool"), nil
		}
		output := "json"

		if args, ok := req.Params.Arguments.(map[string]any); ok {
			for k, v := range args {
				vars[k] = fmt.Sprintf("%v", v)
			}
		}
		stackCopy := *stackMap[stack.Name]
		stackCopy.Vars = executehandler.MergeMaps(stackCopy.Vars, vars)
		executor := &conf.Executor{
			Registry:  nil,
			Config:    nil,
			SourceDir: sourceDir,
		}
		result := executehandler.ExecuteStack(executor, &stackCopy, &output)
		jsondata, _ := json.Marshal(result)
		if auditLogger != nil {
			if err := auditLogger.Write(audit.Entry{
				Stack:      stack.Name,
				Identity:   flattenIdentity(identityFromContext(ctx), excludeidentityheaders),
				Vars:       vars,
				Success:    result.Success,
				Output:     result.Output,
				DurationMs: time.Since(start).Milliseconds(),
				Error:      result.Error,
			}); err != nil {
				log.Printf("audit log write failed: %v", err)
			}
		}
		log.Printf("Tool execution completed: tool: %s", stack.Name)
		return mcp.NewToolResultText(string(jsondata)), nil
	})
}
