package server

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/TechXploreLabs/seristack/internal/audit"
	conf "github.com/TechXploreLabs/seristack/internal/config"
	apperrors "github.com/TechXploreLabs/seristack/internal/errors"
	"github.com/TechXploreLabs/seristack/internal/executehandler"
)

const (
	readHeaderTimeout = 30 * time.Second
	readTimeout       = 30 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

type ErrorResponse struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Details      string `json:"details,omitempty"`
	Timestamp    string `json:"timestamp"`
	RequestID    string `json:"request_id,omitempty"`
}

func Server(config *conf.Config, port *string, addr *string, auditLogger *audit.Logger, identityHeaders map[string]string) error {
	sourceDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	mux := http.NewServeMux()
	hasRoutes := false
	var registeredPatterns = make(map[string]bool)
	stackMap := executehandler.Stackmap(config.Stacks)
	for _, stack := range config.Stacks {
		if stack.Method != "" {
			pattern := stack.Name
			if stack.UrlPath != "" {
				pattern = stack.UrlPath
			}
			if registeredPatterns[pattern] {
				return fmt.Errorf("duplicate route registration: pattern %q is already registered or urlPath already resgistered", stack.Name)
			}
			RegisterHandler(mux, stack, stackMap, sourceDir, auditLogger, identityHeaders)
			hasRoutes = true
			registeredPatterns[pattern] = true
		}
	}
	if !hasRoutes {
		return fmt.Errorf("No endpoint to serve")
	}
	var handler http.Handler = mux
	handler = recoveryMiddleware(handler)
	server := &http.Server{
		Addr:              *addr + ":" + *port,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      0,
		IdleTimeout:       idleTimeout,
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		fmt.Printf("Server starting on http://%s:%s\n", *addr, *port)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !stderrors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server listen error: %w", err)
		}
		return nil
	case <-shutdownCtx.Done():
		stop()
		fmt.Println("Shutting down server gracefully...")

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}

		if err := <-serverErr; err != nil && !stderrors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server listen error: %w", err)
		}

		fmt.Println("Server stopped")
		return nil
	}
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered",
					"error", rec, "method", r.Method, "path", r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func checkAccess(r *http.Request, rules []conf.AccessRule, matchAccess string) bool {
	if len(rules) == 0 {
		return true
	}

	if strings.ToUpper(matchAccess) == "ALL" {
		for _, rule := range rules {
			userValues := splitHeader(r.Header.Get(rule.HeaderName))
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
		userValues := splitHeader(r.Header.Get(rule.HeaderName))
		for _, allowed := range rule.HeaderValue {
			if slices.Contains(userValues, allowed) {
				return true
			}
		}
	}
	return false
}

func splitHeader(val string) []string {
	var out []string
	for _, s := range strings.Split(val, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func extractIdentity(r *http.Request, identityHeaders map[string]string) map[string]string {
	if len(identityHeaders) == 0 {
		return nil
	}
	identity := make(map[string]string)
	for key, headerName := range identityHeaders {
		if val := r.Header.Get(headerName); val != "" {
			identity[key] = val
		}
	}
	return identity
}

func RegisterHandler(mux *http.ServeMux, stack conf.Stack, stackMap map[string]*conf.Stack, sourceDir string, auditLogger *audit.Logger,
	identityHeaders map[string]string) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		output := "yaml"
		requestID := r.Header.Get("X-Request-ID")

		if stack.Method != "" && r.Method != strings.ToUpper(stack.Method) {

			errorResponse := ErrorResponse{
				ErrorCode:    apperrors.METHOD_NOT_ALLOWED.String(),
				ErrorMessage: fmt.Sprintf("Method not allowed. Expected %s", stack.Method),
				Timestamp:    time.Now().Format(time.RFC3339),
				RequestID:    requestID,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
				slog.Error("Method not found", "error", err)
			}
			slog.Error("request failed",
				"requestID", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"errorMessage", errorResponse.ErrorMessage)
			return
		}

		if !checkAccess(r, stack.Access, stack.MatchAccess) {
			errorResponse := ErrorResponse{
				ErrorCode:    apperrors.FORBIDDEN.String(),
				ErrorMessage: fmt.Sprintf("access denied: insufficient permissions to execute this stack %s", stack.Method),
				Timestamp:    time.Now().Format(time.RFC3339),
				RequestID:    requestID,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
				slog.Error("access denied: insufficient permissions to execute this stack", "error", err)
			}
			slog.Error("request failed",
				"requestID", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"errorMessage", errorResponse.ErrorMessage)
			return
		}

		if err := r.ParseForm(); err != nil {

			errorResponse := ErrorResponse{
				ErrorCode:    apperrors.BAD_REQUEST.String(),
				ErrorMessage: fmt.Sprintf("Failed to parse form: %v", err),
				Timestamp:    time.Now().Format(time.RFC3339),
				RequestID:    requestID,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
				slog.Error("failed to write json error response", "error", err)
			}
			slog.Error("request failed",
				"requestID", requestID, "method", r.Method, "path", r.URL.Path, "errorMessage", errorResponse.ErrorMessage)
			return
		}
		executor := &conf.Executor{
			Registry:  nil,
			Config:    nil,
			SourceDir: sourceDir,
		}
		vars := substituteVars(r)
		stackCopy := *stackMap[stack.Name]
		stackCopy.Vars = executehandler.MergeMaps(stackCopy.Vars, vars)
		result := executehandler.ExecuteStack(executor, &stackCopy, &output)

		if auditLogger != nil {
			if err := auditLogger.Write(audit.Entry{
				RequestID:  requestID,
				Stack:      stack.Name,
				Path:       r.URL.Path,
				Method:     r.Method,
				SourceIP:   r.RemoteAddr,
				Identity:   extractIdentity(r, identityHeaders),
				Vars:       vars,
				Success:    result.Success,
				DurationMs: time.Since(start).Milliseconds(),
				Error:      result.Error,
			}); err != nil {
				slog.Error("failed to write audit log entry", "error", err)
			}
		}

		jsondata, _ := json.Marshal(result)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(jsondata); err != nil {
			slog.Error("failed to write response body", "error", err)
		}

		slog.Info("request successful",
			"requestID", requestID, "method", r.Method, "path", r.URL.Path, "query", r.URL.Query())
	}
	if stack.UrlPath == "" {
		mux.HandleFunc("/"+stack.Name, handler)
		fmt.Printf("Registered: %s /%s\n", stack.Method, stack.Name)
	} else {
		mux.HandleFunc(stack.UrlPath, handler)
		fmt.Printf("Registered: %s %s\n", stack.Method, stack.UrlPath)
	}

}

func substituteVars(r *http.Request) map[string]string {
	vars := make(map[string]string)
	for key, values := range r.URL.Query() {
		vars[key] = values[len(values)-1]
	}
	for key, values := range r.PostForm {
		vars[key] = values[len(values)-1]
	}
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		body, err := io.ReadAll(r.Body)
		if err == nil {
			r.Body = io.NopCloser(bytes.NewBuffer(body))
			var jsonVars map[string]any
			if err := json.Unmarshal(body, &jsonVars); err == nil {
				for k, v := range jsonVars {
					vars[k] = fmt.Sprintf("%v", v)
				}
			}
		}
	}
	for key, values := range r.Header {
		if strings.HasPrefix(key, "X-") && len(values) > 0 {
			vars[key] = values[0]
		}
	}
	return vars
}
