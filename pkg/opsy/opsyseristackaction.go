package opsyseristackaction

import (
	"fmt"
	"time"

	"github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/trigger"
	yaml "gopkg.in/yaml.v3"
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
	Config    *config.Config
	StackName string
	Vars      map[string]string
	Format    string
}

func NewConfigFromYAML(data []byte) (*config.Config, error) {
	var stackDef config.Config
	if err := yaml.Unmarshal(data, &stackDef); err != nil {
		return &config.Config{}, fmt.Errorf("failed to decode stack config: %w", err)
	}

	return &stackDef, nil
}

func OpsySeristack(conf *Config) (Result, error) {
	config, err := trigger.SingleStackCheck(conf.Config, &conf.StackName)
	if err != nil {
		return Result{}, fmt.Errorf("%w", err)
	}
	output := conf.Format
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
