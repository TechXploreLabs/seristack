package opsyseristackaction

import (
	"fmt"
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
	ConfigFile string
	StackName  string
	Vars       map[string]string
}

func OpsySeristack(conf Config) (Result, error) {
	config, err := config.LoadConfig(conf.ConfigFile)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load config: %w", err)
	}
	config, err = trigger.SingleStackCheck(config, &conf.StackName)
	if err != nil {
		return Result{}, fmt.Errorf("%w", err)
	}
	output := "yaml"
	result := trigger.RunTrigger(config, &output, &conf.Vars)
	actionResult := Result{
		Name:            result[0].Name,
		Success:         result[0].Success,
		Output:          result[0].Output,
		Error:           result[0].Error,
		Duration:        result[0].Duration,
		ContinueOnError: result[0].ContinueOnError,
	}
	return actionResult, nil
}
