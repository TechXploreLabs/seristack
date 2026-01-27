package registry

import (
	"hash/fnv"
	"slices"

	"github.com/TechXploreLabs/seristack/internal/config"
)

func NewRegistry(order *[][]string) *config.Registry {
	var batch_length []int
	for _, each_batch_count := range *order {
		batch_length = append(batch_length, len(each_batch_count))
	}
	slices.SortFunc(batch_length, func(a, b int) int {
		return b - a
	})
	shardCount := calculateOptimalShards(batch_length[0])
	r := &config.Registry{
		Shards:     make([]*config.Shard, shardCount),
		ShardCount: uint32(shardCount),
	}
	for i := range r.Shards {
		r.Shards[i] = &config.Shard{
			Results: make(map[string]*config.Result),
			Vars:    make(map[string]interface{}),
		}
	}
	return r
}
func calculateOptimalShards(stackCount int) int {
	if stackCount <= 0 {
		return 2
	} else if stackCount < 4 {
		return 4
	} else if stackCount < 8 {
		return 8
	} else if stackCount < 16 {
		return 16
	} else if stackCount < 32 {
		return 32
	} else if stackCount < 64 {
		return 64
	} else if stackCount < 128 {
		return 128
	} else if stackCount < 256 {
		return 256
	} else {
		return 512
	}
}

func getShard(r *config.Registry, key string) *config.Shard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return r.Shards[h.Sum32()%r.ShardCount]
}

func Set(r *config.Registry, name string, result *config.Result) {
	s := getShard(r, name)
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.Results[name] = result
}

func GetAllVars(r *config.Registry) map[string]string {
	allVars := make(map[string]string)
	for _, shard := range r.Shards {
		shard.Mu.RLock()
		for stackName, result := range shard.Results {
			allVars[stackName] = result.Output
		}
		shard.Mu.RUnlock()
	}
	return allVars
}
