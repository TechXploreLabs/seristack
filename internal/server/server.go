package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	conf "github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/executehandler"
	"github.com/TechXploreLabs/seristack/internal/function"
	"github.com/TechXploreLabs/seristack/internal/shellexecutor"
)

func Server(config *conf.Config) {
	mux := http.NewServeMux()
	stackMap := executehandler.Stackmap(config.Stacks)
	for _, endpoint := range config.Server.Endpoints {
		RegisterHandler(mux, endpoint, stackMap)
	}
	port := config.Server.Port
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server starting on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func RegisterHandler(mux *http.ServeMux, endpoint conf.Endpoint, stackMap map[string]*conf.Stack) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if endpoint.Method != "" && r.Method != strings.ToUpper(endpoint.Method) {
			http.Error(w, fmt.Sprintf("Method not allowed. Expected %s", endpoint.Method), http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
			return
		}
		vars := substituteVars(r)

		shell, shellArg, err := shellexecutor.Shellargs(stackMap[endpoint.Stackname].Shell, stackMap[endpoint.Stackname].ShellArg)
		if err != nil {
			http.Error(w, fmt.Sprintf("Shell configuration error: %v", err), http.StatusInternalServerError)
			log.Printf("Shell config error for %s: %v", endpoint.Path, err)
			return
		}
		var allOutput bytes.Buffer
		for _, cmd := range stackMap[endpoint.Stackname].Cmds {
			cmd, err := function.ReplaceVariables(vars, nil, cmd)
			if err != nil {
				http.Error(w, fmt.Sprintf("Variable Subtitution failed: %s", err), http.StatusInternalServerError)
				log.Printf("Variable Subtitution failed %s", err)
				return
			}
			output, err := shellexecutor.ShellExec(stackMap[endpoint.Stackname].WorkDir, shell, shellArg, cmd)
			if err != nil {
				http.Error(w, fmt.Sprintf("Command execution failed: %v", err), http.StatusInternalServerError)
				log.Printf("Error executing command for %s: %v", endpoint.Path, err)
				return
			}
			allOutput.Write(output)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write(allOutput.Bytes())
		log.Printf("%s %s - Params: %v - Success",
			r.Method, endpoint.Path, r.URL.Query())
	}
	mux.HandleFunc(endpoint.Path, handler)
	fmt.Printf("Registered: %s %s\n", endpoint.Method, endpoint.Path)
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
			r.Body = io.NopCloser(bytes.NewBuffer(body)) // Restore body

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
