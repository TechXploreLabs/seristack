package opsyseristackaction

import (
	"fmt"
	"os"
	"time"

	"github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/trigger"
)

type Result struct {
	Name            string
	Success         bool
	Output          string
	Error           string
	Duration        time.Duration
	ContinueOnError bool
}

type Config struct {
	configfile string
	stackname  string
	vars       map[string]string
}

func OpsySeristack(conf Config) Result {
	config, err := config.LoadConfig(conf.configfile)
	if err != nil {
		fmt.Printf("failed to load config: %v", err)
		os.Exit(1)
	}
	config = trigger.SingleStackCheck(config, &conf.stackname)
	config.Stacks[0].Vars = conf.vars
	output := "yaml"
	result := trigger.RunTrigger(config, &output)
	actionResult := Result{
		Name:            result[0].Name,
		Success:         result[0].Success,
		Output:          result[0].Output,
		Error:           result[0].Error,
		Duration:        result[0].Duration,
		ContinueOnError: result[0].ContinueOnError,
	}
	return actionResult
}
