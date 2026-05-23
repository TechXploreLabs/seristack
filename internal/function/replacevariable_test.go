package function

import (
	"testing"

	"github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/registry"
)

func TestReplaceVariables_UsesOnlyReferencedResultKeys(t *testing.T) {
	r := &config.Registry{
		Shards:     make([]*config.Shard, 4),
		ShardCount: 4,
	}
	for i := range r.Shards {
		r.Shards[i] = &config.Shard{Results: map[string]*config.Result{}}
	}

	registry.Set(r, "stack1", &config.Result{Name: "stack1", Output: "out-1"})
	registry.Set(r, "stack2", &config.Result{Name: "stack2", Output: "out-2"})
	registry.Set(r, "stack3", &config.Result{Name: "stack3", Output: "out-3"})

	executor := &config.Executor{Registry: r}
	input := `echo {{.Result.stack2}}`

	got, err := ReplaceVariables(nil, executor, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "echo out-2" {
		t.Fatalf("expected only stack2 substitution, got %q", got)
	}
}

func TestCollectResultKeys(t *testing.T) {
	input := `echo {{.Result.stack1}} && echo {{ .Result.stack2 }}`
	keys := collectResultKeys(input)

	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d (%v)", len(keys), keys)
	}

	seen := map[string]bool{}
	for _, key := range keys {
		seen[key] = true
	}

	if !seen["stack1"] || !seen["stack2"] {
		t.Fatalf("expected stack1 and stack2 keys, got %v", keys)
	}
}
