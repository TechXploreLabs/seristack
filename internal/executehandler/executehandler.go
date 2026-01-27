package executehandler

import (
	"fmt"
	"strings"
	"sync"

	"github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/registry"
)

func Stackmap(e []config.Stack) map[string]*config.Stack {
	stackMap := make(map[string]*config.Stack, len(e))
	for i := range e {
		stackMap[e[i].Name] = &e[i]
	}
	return stackMap
}

func Execute(e *config.Executor, order *[][]string) error {
	stackMap := Stackmap(e.Config.Stacks)

	for batchNum, batch := range *order {
		fmt.Printf("\n%s\n", strings.Repeat("=", 80))
		fmt.Printf("BATCH %d: %v\n", batchNum+1, batch)
		fmt.Printf("%s\n", strings.Repeat("=", 80))

		var wg sync.WaitGroup
		errChan := make(chan error, len(batch))
		for _, stackName := range batch {
			wg.Add(1)
			stack := stackMap[stackName]
			go func(s *config.Stack) {
				defer wg.Done()
				result := ExecuteStack(e, s)
				registry.Set(e.Registry, s.Name, result)
				fmt.Printf(`
########################
stack: %s
output:
%s
########################
`, s.Name, result.Output)
				if !result.Success && !s.ContinueOnError {
					errChan <- fmt.Errorf("stack '%s' failed: %v", s.Name, result.Error)
				}
			}(stack)
		}

		wg.Wait()
		close(errChan)
		for err := range errChan {
			return err
		}
	}
	return nil
}
