// Unit tests for ExecutionOrder
package function

import (
	"reflect"
	"testing"

	"github.com/TechXploreLabs/seristack/internal/config"
)

func TestExecutionOrder(t *testing.T) {
	// Basic test: two independent stacks
	stacks := []config.Stack{
		{Name: "s1"},
		{Name: "s2"},
	}
	order, err := ExecutionOrder(stacks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both stacks should be scheduled independently (order[0] contains both names, in any order)
	found := map[string]bool{"s1": false, "s2": false}
	for _, name := range order[0] {
		found[name] = true
	}
	if !found["s1"] || !found["s2"] || len(order[0]) != 2 {
		t.Errorf("expected both s1 and s2 at the same level, got %+v", order)
	}

	// Test a dependency chain: s1 -> s2
	stacks = []config.Stack{
		{Name: "s1"},
		{Name: "s2", DependsOn: []string{"s1"}},
	}
	order, err = ExecutionOrder(stacks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(reflect.DeepEqual(order[0], []string{"s1"}) && reflect.DeepEqual(order[1], []string{"s2"})) {
		t.Errorf("expected [[s1],[s2]], got %+v", order)
	}

	// Circular dependency should return an error
	stacks = []config.Stack{
		{Name: "s1", DependsOn: []string{"s2"}},
		{Name: "s2", DependsOn: []string{"s1"}},
	}
	_, err = ExecutionOrder(stacks)
	if err == nil {
		t.Errorf("expected error for circular dependency, got nil")
	}
}
