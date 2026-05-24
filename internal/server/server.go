package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	conf "github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/executehandler"
)

func Server(config *conf.Config, port *string, addr *string, skip *bool) error {
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
	if hasRoutes {
		fmt.Printf("Server starting on http://%s:%s\n", *addr, *port)
		if err := http.ListenAndServe(*addr+":"+*port, mux); err != nil {
			return fmt.Errorf("server listen error: %w", err)
		}
	} else {
		return fmt.Errorf("No endpoint to serve")
	}
	return nil
}

func RegisterHandler(mux *http.ServeMux, stack conf.Stack, stackMap map[string]*conf.Stack) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		sourceDir, _ := os.Getwd()
		output := "yaml"
		if stack.Method != "" && r.Method != strings.ToUpper(stack.Method) {
			http.Error(w, fmt.Sprintf("Method not allowed. Expected %s", stack.Method), http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
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
		log.Printf("%s %s - Params: %v - Success",
			r.Method, stack.Name, r.URL.Query())
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
