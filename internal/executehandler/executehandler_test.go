// Unit tests for executehandler.go
package executehandler

import (
	"reflect"
	"testing"

	"github.com/TechXploreLabs/seristack/internal/config"
)

func TestStackmap(t *testing.T) {
	stacks := []config.Stack{
		{Name: "stack1"},
		{Name: "stack2"},
	}

	got := Stackmap(stacks)
	if len(got) != 2 {
		t.Errorf("expected map of 2, got %d", len(got))
	}
	if got["stack1"] == nil || got["stack1"].Name != "stack1" {
		t.Errorf("missing or incorrect entry for 'stack1'")
	}
	if got["stack2"] == nil || got["stack2"].Name != "stack2" {
		t.Errorf("missing or incorrect entry for 'stack2'")
	}

	// Edge case: empty input
	stacks = []config.Stack{}
	got = Stackmap(stacks)
	if !reflect.DeepEqual(got, map[string]*config.Stack{}) {
		t.Errorf("expected empty map, got %v", got)
	}
}
