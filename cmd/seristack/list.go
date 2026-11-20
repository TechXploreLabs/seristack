package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	conf "github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/function"
)

var (
	listOrder bool
	listLong  bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the stacks in the config file",
	Long: `
  List the stacks in the --config file.

  seristack list -c config.yaml
  seristack list -c config.yaml --order
  seristack list -c config.yaml --long

  --order shows the execution order. Stacks in the same batch run concurrently.
  It also fails on a dependency cycle or an unknown dependency, without
  running anything.`,
	RunE: list,
}

func init() {
	listCmd.Flags().BoolVar(&listOrder, "order", false, "show execution order (stacks in one batch run concurrently)")
	listCmd.Flags().BoolVar(&listLong, "long", false, "show details for each stack")
	rootCmd.AddCommand(listCmd)
}

type stackInfo struct {
	Name        string   `json:"name"`
	Batch       int      `json:"batch"`
	Description string   `json:"description,omitempty"`
	HTTP        string   `json:"http,omitempty"`
	MCP         bool     `json:"mcp"`
	DependsOn   []string `json:"dependsOn,omitempty"`
	Mode        string   `json:"executionMode"`
	Count       int      `json:"count"`
	Timeout     string   `json:"timeout,omitempty"`
}

func list(cmd *cobra.Command, args []string) error {
	config, err := conf.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("%s", color.RedString("Error: [failed to load config], %v", err))
	}

	order, err := function.ExecutionOrder(config.Stacks)
	if err != nil {
		return fmt.Errorf("%s", color.RedString("Error: [dependency resolution failed], %v", err))
	}

	if !listOrder && !listLong {
		for _, s := range config.Stacks {
			fmt.Println(s.Name)
		}
		return nil
	}

	if listOrder && !listLong {
		for i, batch := range order {
			fmt.Printf("batch %d: %s\n", i+1, strings.Join(batch, ", "))
		}
		return nil
	}

	byName := make(map[string]*conf.Stack, len(config.Stacks))
	for i := range config.Stacks {
		byName[config.Stacks[i].Name] = &config.Stacks[i]
	}

	var infos []stackInfo
	for i, batch := range order {
		for _, name := range batch {
			s := byName[name]
			if s == nil {
				continue
			}
			mode := s.ExecutionMode
			if mode == "" {
				mode = "PARALLEL"
			}
			info := stackInfo{
				Name:        s.Name,
				Batch:       i + 1,
				Description: s.Description,
				MCP:         s.Description != "",
				DependsOn:   s.DependsOn,
				Mode:        mode,
				Count:       s.Count,
				Timeout:     s.Timeouts,
			}
			if s.Method != "" {
				path := "/" + s.Name
				if s.UrlPath != "" {
					path = s.UrlPath
				}
				info.HTTP = strings.ToUpper(s.Method) + " " + path
			}
			infos = append(infos, info)
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "BATCH\tNAME\tEXPOSED AS\tDEPENDS ON\tMODE\tCOUNT\tTIMEOUT")
	for _, s := range infos {
		exposed := []string{"cli"}
		if s.HTTP != "" {
			exposed = append(exposed, s.HTTP)
		}
		if s.MCP {
			exposed = append(exposed, "mcp")
		}
		deps := "-"
		if len(s.DependsOn) > 0 {
			deps = strings.Join(s.DependsOn, ",")
		}
		count := fmt.Sprint(s.Count)
		if s.Count == 0 {
			count = "0 (skipped)"
		}
		timeout := s.Timeout
		if timeout == "" {
			timeout = "default"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			s.Batch, s.Name, strings.Join(exposed, " "), deps, s.Mode, count, timeout)
	}
	return w.Flush()
}
