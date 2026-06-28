package server

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	conf "github.com/TechXploreLabs/seristack/internal/config"
	apperrors "github.com/TechXploreLabs/seristack/internal/errors"
	"github.com/TechXploreLabs/seristack/internal/executehandler"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
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

func Server(config *conf.Config, port *string, addr *string) error {
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
			RegisterHandler(mux, stack, stackMap)
			hasRoutes = true
			registeredPatterns[pattern] = true
		}
	}
	if !hasRoutes {
		return fmt.Errorf("No endpoint to serve")
	}

	server := &http.Server{
		Addr:              *addr + ":" + *port,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
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

func RegisterHandler(mux *http.ServeMux, stack conf.Stack, stackMap map[string]*conf.Stack) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		sourceDir, _ := os.Getwd()
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
			json.NewEncoder(w).Encode(errorResponse)
			log.Printf("RequestID: %s, Method: %s, Path: %s, Error: %s",
				requestID, r.Method, r.URL.Path, errorResponse.ErrorMessage)
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
			json.NewEncoder(w).Encode(errorResponse)
			log.Printf("RequestID: %s, Method: %s, Path: %s, Error: %s",
				requestID, r.Method, r.URL.Path, errorResponse.ErrorMessage)
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
		jsondata, _ := json.Marshal(result)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsondata)

		log.Printf("RequestID: %s, Method: %s, Path: %s, Params: %v, Status: Success",
			requestID, r.Method, r.URL.Path, r.URL.Query())
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
			var jsonVars map[string]interface{}
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
