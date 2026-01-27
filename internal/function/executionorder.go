package function

import (
	"fmt"

	"github.com/TechXploreLabs/seristack/internal/config"
)

func ExecutionOrder(stacks []config.Stack) ([][]string, error) {
	n := len(stacks)
	inDegree := make(map[string]int, n)
	dependents := make(map[string][]string, n)
	for _, s := range stacks {
		if _, exists := inDegree[s.Name]; exists {
			return nil, fmt.Errorf("duplicate stack name '%s'", s.Name)
		}
		inDegree[s.Name] = 0
	}
	for _, s := range stacks {
		for _, dep := range s.DependsOn {
			if _, exists := inDegree[dep]; !exists {
				return nil, fmt.Errorf("dependency '%s' not found for stack '%s'", dep, s.Name)
			}
			if dep == s.Name {
				return nil, fmt.Errorf("stack '%s' depends on itself", s.Name)
			}
			dependents[dep] = append(dependents[dep], s.Name)
			inDegree[s.Name]++
		}
	}
	var order [][]string
	processed := 0
	var currentBatch []string
	for name, degree := range inDegree {
		if degree == 0 {
			currentBatch = append(currentBatch, name)
		}
	}
	for len(currentBatch) > 0 {
		order = append(order, currentBatch)
		processed += len(currentBatch)

		var nextBatch []string
		for _, name := range currentBatch {
			for _, dependent := range dependents[name] {
				inDegree[dependent]--
				if inDegree[dependent] == 0 {
					nextBatch = append(nextBatch, dependent)
				}
			}
		}
		currentBatch = nextBatch
	}
	if processed != n {
		return nil, fmt.Errorf("cyclic dependency detected")
	}
	return order, nil
}
