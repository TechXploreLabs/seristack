package executehandler

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"maps"

	"github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/registry"
	"github.com/TechXploreLabs/seristack/internal/shellexecutor"
	"github.com/fatih/color"
)

// Outputs stackname and stack details in dict format

func Stackmap(e []config.Stack) map[string]*config.Stack {
	stackMap := make(map[string]*config.Stack, len(e))
	for i := range e {
		stackMap[e[i].Name] = &e[i]
	}
	return stackMap
}

// Vars override

func MergeMaps(base, override map[string]string) map[string]string {
	result := make(map[string]string)
	maps.Copy(result, base)
	maps.Copy(result, override)
	return result
}

// Function that call execute stack

func Execute(e *config.Executor, order *[][]string, output *string, varsMap *map[string]string) []*config.Result {
	stackMap := Stackmap(e.Config.Stacks)
	var consolidatedresult []*config.Result
	for _, batch := range *order {
		var wg sync.WaitGroup
		resultChan := make(chan *config.Result, len(batch))
		for _, stackName := range batch {
			wg.Add(1)
			stack := stackMap[stackName]
			go func(stack *config.Stack, output *string) {
				defer wg.Done()
				if varsMap != nil {
					stack.Vars = MergeMaps(stack.Vars, *varsMap)
				}
				result := ExecuteStack(e, stack, output)
				if e.Registry != nil {
					registry.Set(e.Registry, stack.Name, result)
				}
				resultChan <- result
			}(stack, output)
		}
		wg.Wait()
		close(resultChan)
		if *output != "" {
			for value := range resultChan {
				consolidatedresult = append(consolidatedresult, value)
			}
		}
	}
	return consolidatedresult
}

// Function for executing stack

func ExecuteStack(e *config.Executor, stack *config.Stack, output *string) *config.Result {
	start := time.Now()
	var result *config.Result
	result = shellexecutor.ExecuteShell(e, stack)
	result.Duration = time.Since(start)
	if *output == "" {
		fmt.Printf(`
stack: %s
continueOnError: %t
duration: %.2fs
success: %t
`, stack.Name, result.ContinueOnError, result.Duration.Seconds(), result.Success)
		if result.Error == "" {
			color.Green(`
output:
┌─
`)
			for _, line := range strings.Split(strings.TrimSpace(result.Output), "\n") {
				color.Green("│ %s\n", line)
			}
			color.Green("└─\n")
		}
		if result.Error != "" {
			color.Yellow(`
output:
┌─
`)
			for _, line := range strings.Split(strings.TrimSpace(result.Output), "\n") {
				color.Yellow("│ %s\n", line)
			}
			color.Yellow("└─\n")
			color.Red(`
error:
┌─
`)
			for _, line := range strings.Split(strings.TrimSpace(result.Error), "\n") {
				color.Red("│ %s\n", line)
			}
			color.Red("└─\n")
		}
	}
	return result
}
