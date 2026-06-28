package trigger

import (
	"fmt"
	"os"

	conf "github.com/TechXploreLabs/seristack/internal/config"
	exe "github.com/TechXploreLabs/seristack/internal/executehandler"
	"github.com/TechXploreLabs/seristack/internal/function"
	"github.com/TechXploreLabs/seristack/internal/registry"
)

func SingleStackCheck(config *conf.Config, stack *string) (*conf.Config, error) {
	stackmap := exe.Stackmap(config.Stacks)
	if newstack, ok := stackmap[*stack]; ok {
		newstack.DependsOn = nil
		config = &conf.Config{
			Stacks: []conf.Stack{*newstack},
		}
	} else {
		return nil, fmt.Errorf("Stack not exist.")
	}
	return config, nil
}

func RunTrigger(config *conf.Config, output *string, varsMap *map[string]string) ([]*conf.Result, error) {
	order, err := function.ExecutionOrder(config.Stacks)
	if err != nil {
		return nil, fmt.Errorf("dependency resolution failed due to %v", err)

	}
	sourceDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory %v", err)
	}
	executor := &conf.Executor{
		Registry:  registry.NewRegistry(&order),
		Config:    config,
		SourceDir: sourceDir,
	}
	switch o := *output; o {
	case "":
		exe.Execute(executor, &order, output, varsMap)
		return nil, nil
	case "yaml", "json":
		consolidatedresult := exe.Execute(executor, &order, output, varsMap)
		return consolidatedresult, nil
	}
	return nil, nil
}
