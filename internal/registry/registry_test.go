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

func TestDelete_RemovesOnlyRequestedStacks(t *testing.T) {
	r := newTestRegistry(8)

	Set(r, "build", &config.Result{Name: "build", Output: "build-output"})
	Set(r, "test", &config.Result{Name: "test", Output: "test-output"})
	Set(r, "deploy", &config.Result{Name: "deploy", Output: "deploy-output"})

	Delete(r, []string{"build", "test"})

	allVars := GetAllVars(r)
	if _, ok := allVars["build"]; ok {
		t.Fatalf("expected build output to be discarded")
	}
	if _, ok := allVars["test"]; ok {
		t.Fatalf("expected test output to be discarded")
	}
	if got, ok := allVars["deploy"]; !ok || got != "deploy-output" {
		t.Fatalf("expected deploy output to be retained, got: %q", got)
	}
}

func TestDelete_NoOpForNilRegistryOrEmptyNames(t *testing.T) {
	Delete(nil, []string{"any"})

	r := newTestRegistry(4)
	Set(r, "stack1", &config.Result{Name: "stack1", Output: "out1"})

	Delete(r, nil)
	Delete(r, []string{})
	Delete(r, []string{""})

	allVars := GetAllVars(r)
	if got, ok := allVars["stack1"]; !ok || got != "out1" {
		t.Fatalf("expected stack1 output to remain unchanged, got: %q", got)
	}
}
