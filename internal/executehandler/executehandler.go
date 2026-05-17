package executehandler

import (
	"fmt"
	"regexp"
	"slices"
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
					registry.Delete(e.Registry, stack.DiscardOutput)
				}
				resultChan <- result
			}(stack, output)
		}
		wg.Wait()
		close(resultChan)
		batchFailed := false
		for result := range resultChan {
			consolidatedresult = append(consolidatedresult, result)
			if !result.Success && !result.ContinueOnError {
				batchFailed = true
			}
		}
		if batchFailed {
			break
		}
	}
	if *output != "" {
		return consolidatedresult
	}
	return nil
}

// Function for executing stack

func ExecuteStack(e *config.Executor, stack *config.Stack, output *string) *config.Result {
	start := time.Now()
	if err := ValidateStackVars(stack); err != nil {
		if *output == "" {
			color.Red("\nstack: %s\nvalidation_error: %s\n", stack.Name, err.Error())
		}
		return &config.Result{
			Name:            stack.Name,
			Success:         false,
			Error:           err.Error(),
			Duration:        time.Since(start),
			ContinueOnError: false,
		}
	}

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

func ValidateStackVars(stack *config.Stack) error {
	if stack == nil || len(stack.VarRules) == 0 {
		return nil
	}

	for varName, rule := range stack.VarRules {
		value, ok := stack.Vars[varName]
		if !ok {
			return fmt.Errorf("variable validation failed: variable '%s' is required", varName)
		}

		if len(rule.AllowedValue) > 0 {
			if !slices.Contains(rule.AllowedValue, value) {
				return fmt.Errorf("variable validation failed for '%s': value '%s' must be one of %v", varName, value, rule.AllowedValue)
			}
		}

		if len(rule.DeniedValue) > 0 {
			if slices.Contains(rule.DeniedValue, value) {
				return fmt.Errorf("variable validation failed for '%s': value '%s' is denied", varName, value)
			}
		}

		if strings.TrimSpace(rule.AllowedRegex) != "" {
			pattern, err := extractRegexPattern(rule.AllowedRegex)
			if err != nil {
				return fmt.Errorf("variable validation failed for '%s': %w", varName, err)
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("variable validation failed for '%s': invalid allowed_regex '%s': %w", varName, pattern, err)
			}
			if !re.MatchString(value) {
				return fmt.Errorf("variable validation failed for '%s': value '%s' does not match allowed_regex '%s'", varName, value, pattern)
			}
		}

		if strings.TrimSpace(rule.DeniedRegex) != "" {
			pattern, err := extractRegexPattern(rule.DeniedRegex)
			if err != nil {
				return fmt.Errorf("variable validation failed for '%s': %w", varName, err)
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("variable validation failed for '%s': invalid denied_regex '%s': %w", varName, pattern, err)
			}
			if re.MatchString(value) {
				return fmt.Errorf("variable validation failed for '%s': value '%s' matches denied_regex '%s'", varName, value, pattern)
			}
		}
	}

	return nil
}

func extractRegexPattern(regexRule string) (string, error) {
	regexRule = strings.TrimSpace(regexRule)
	if !strings.HasPrefix(regexRule, "regex(") || !strings.HasSuffix(regexRule, ")") {
		return "", fmt.Errorf("invalid regex rule '%s': expected regex(...) format", regexRule)
	}

	pattern := strings.TrimPrefix(regexRule, "regex(")
	pattern = strings.TrimSuffix(pattern, ")")
	pattern = strings.TrimSpace(pattern)
	pattern = strings.Trim(pattern, "\"")
	pattern = strings.Trim(pattern, "'")
	if pattern == "" {
		return "", fmt.Errorf("invalid regex rule '%s': empty pattern", regexRule)
	}
	return pattern, nil
}
