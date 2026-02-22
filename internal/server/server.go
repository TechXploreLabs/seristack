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
	"gopkg.in/yaml.v3"
)

func Server(config *conf.Config) {
	mux := http.NewServeMux()
	stackMap := executehandler.Stackmap(config.Stacks)
	for _, endpoint := range config.Server.Endpoints {
		RegisterHandler(mux, endpoint, stackMap)
	}
	port := config.Server.Port
	host := config.Server.Host
	if port == "" {
		port = "8080"
	}
	if host == "" {
		host = "127.0.0.1"
	}
	fmt.Printf("Server starting on http://%s:%s\n", host, port)
	if err := http.ListenAndServe(host+":"+port, mux); err != nil {
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
		sourceDir, _ := os.Getwd()
		executor := &conf.Executor{
			Registry:  nil,
			Config:    nil,
			SourceDir: sourceDir,
		}
		vars := substituteVars(r)
		stackMap[endpoint.Stackname].Vars = executehandler.MergeMaps(stackMap[endpoint.Stackname].Vars, vars)
		output := "yaml"
		result := executehandler.ExecuteStack(executor, stackMap[endpoint.Stackname], &output)
		yamldata, _ := yaml.Marshal(result)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(yamldata)
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
