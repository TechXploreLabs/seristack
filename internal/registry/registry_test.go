package registry

import (
	"testing"

	"github.com/TechXploreLabs/seristack/internal/config"
)

func newTestRegistry(shardCount uint32) *config.Registry {
	r := &config.Registry{
		Shards:     make([]*config.Shard, shardCount),
		ShardCount: shardCount,
	}
	for i := range r.Shards {
		r.Shards[i] = &config.Shard{Results: map[string]*config.Result{}}
	}
	return r
}

func TestGetVarsByNames_ReturnsOnlyRequestedExistingStacks(t *testing.T) {
	r := newTestRegistry(8)

	Set(r, "build", &config.Result{Name: "build", Output: "build-output"})
	Set(r, "test", &config.Result{Name: "test", Output: "test-output"})
	Set(r, "deploy", &config.Result{Name: "deploy", Output: "deploy-output"})

	selected := GetVarsByNames(r, []string{"test", "missing", ""})

	if len(selected) != 1 {
		t.Fatalf("expected only one selected output, got %d (%v)", len(selected), selected)
	}
	if got, ok := selected["test"]; !ok || got != "test-output" {
		t.Fatalf("expected test output to be returned, got: %q", got)
	}
}

func TestGetVarsByNames_NoOpForNilRegistryOrEmptyNames(t *testing.T) {
	selected := GetVarsByNames(nil, []string{"stack1"})
	if len(selected) != 0 {
		t.Fatalf("expected empty map for nil registry, got: %v", selected)
	}

	r := newTestRegistry(4)
	Set(r, "stack1", &config.Result{Name: "stack1", Output: "out1"})

	selected = GetVarsByNames(r, nil)
	if len(selected) != 0 {
		t.Fatalf("expected empty map for nil names, got: %v", selected)
	}

	selected = GetVarsByNames(r, []string{})
	if len(selected) != 0 {
		t.Fatalf("expected empty map for empty names, got: %v", selected)
	}
}
