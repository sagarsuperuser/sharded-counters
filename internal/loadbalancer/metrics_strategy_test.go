package loadbalancer_test

import (
	"testing"

	"sharded-counters/internal/loadbalancer"
	shardmetadata "sharded-counters/internal/shard_metadata"
)

func TestSelectShardByCPU(t *testing.T) {
	strategy := &loadbalancer.MetricsStrategy{}

	// Test Case 1: Normal scenario with multiple shards
	t.Run("SelectShardWithLowestCPU", func(t *testing.T) {
		shards := []*shardmetadata.Shard{
			{ShardID: "shard1", CPUUtilization: 50.0},
			{ShardID: "shard2", CPUUtilization: 20.0},
			{ShardID: "shard3", CPUUtilization: 75.0},
		}

		selectedShard, err := strategy.SelectShard(shards)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if selectedShard.ShardID != "shard2" {
			t.Errorf("Expected shard2, got %s", selectedShard.ShardID)
		}
	})

	// Test Case 2: Empty shard list
	t.Run("EmptyShardList", func(t *testing.T) {
		shards := []*shardmetadata.Shard{}

		_, err := strategy.SelectShard(shards)
		if err == nil {
			t.Fatal("Expected an error, but got none")
		}

		expectedError := "no shard with valid CPU metrics found"
		if err.Error() != expectedError {
			t.Errorf("Expected error %q, got %q", expectedError, err.Error())
		}
	})

	// Test Case 3: All shards have 100% CPU utilization
	t.Run("AllShardsAtMaxCPU", func(t *testing.T) {
		shards := []*shardmetadata.Shard{
			{ShardID: "shard1", CPUUtilization: 100.0},
			{ShardID: "shard2", CPUUtilization: 100.0},
		}

		_, err := strategy.SelectShard(shards)
		if err == nil {
			t.Fatal("Expected an error, but got none")
		}

		expectedError := "no shard with valid CPU metrics found"
		if err.Error() != expectedError {
			t.Errorf("Expected error %q, got %q", expectedError, err.Error())
		}
	})
}
