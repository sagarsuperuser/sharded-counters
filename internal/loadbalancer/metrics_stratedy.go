package loadbalancer

import (
	"fmt"
	shardmetadata "sharded-counters/internal/shard_metadata"
)

// MetricsStrategy selects a shard based on CPU utilization.
type MetricsStrategy struct{}

// SelectShard selects the shard with the least CPU utilization.
func (m *MetricsStrategy) SelectShard(shards []*shardmetadata.Shard) (*shardmetadata.Shard, error) {
	return m.selectShardByCPU(shards)
}

func (m *MetricsStrategy) selectShardByCPU(shards []*shardmetadata.Shard) (*shardmetadata.Shard, error) {
	var selectedShard *shardmetadata.Shard
	var minCPU float64 = 100.0 // Start with a high CPU value.

	for _, shard := range shards {
		if shard.CPUUtilization < minCPU {
			minCPU = shard.CPUUtilization
			selectedShard = shard
		}
	}

	if minCPU == 100.0 {
		return &shardmetadata.Shard{}, fmt.Errorf("no shard with valid CPU metrics found")
	}

	return selectedShard, nil
}
