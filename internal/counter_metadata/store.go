package countermetadata

import (
	"encoding/json"
	"fmt"
	"log"
	"sharded-counters/internal/etcd"
	shardmetadata "sharded-counters/internal/shard_metadata"
)

const CounterPrefix = "counters" // Prefix used to identify counter keys in etcd

// saveCounterMetadata saves counter metadata in Etcd.
func SaveCounterMetadata(manager etcd.Manager, counterID string, shards []*shardmetadata.Shard) error {
	shardIds := GetShardIds(shards)
	data, err := json.Marshal(shardIds)
	if err != nil {
		return fmt.Errorf("failed to marshal shards: %v", err)
	}
	key := fmt.Sprintf("%s/%s", CounterPrefix, counterID)
	err = manager.SaveMetadata(key, string(data))
	if err != nil {
		return fmt.Errorf("failed to store metadata in etcd: %v", err)
	}
	return nil
}

// getCounterMetadata retrieves counter metadata i.e. assigned shards from Etcd.
func GetCounterMetadata(manager etcd.Manager, counterID string) ([]*shardmetadata.Shard, error) {
	// Fetch the metadata from Etcd using the provided key
	key := fmt.Sprintf("%s/%s", CounterPrefix, counterID)
	data, err := manager.Get(key)
	if err != nil {
		return nil, err
	}
	// Unmarshal the JSON data into a slice of strings
	var shards []string
	err = json.Unmarshal([]byte(data), &shards)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %v", err)
	}

	shardsList := GetShardObjList(shards)
	return shardsList, nil

}

func LoadOrStore(etcdManager etcd.Manager, counterID string) ([]*shardmetadata.Shard, error) {
	// Retrieve assigned shards (pods) for counter
	counterShards, metadataErr := GetCounterMetadata(etcdManager, counterID)
	if etcd.IsKeyNotFound(metadataErr) {
		// Retrieve all available shards (pods) from Etcd.
		allAliveShards, err := shardmetadata.GetAliveShards(etcdManager)
		if err != nil {
			return nil, err
		}

		// Assign shards to the counter.
		counterShards = assignShards(allAliveShards)

		// Save metadata to Etcd.
		if err := SaveCounterMetadata(etcdManager, counterID, counterShards); err != nil {
			return nil, err
		}
		log.Printf("Stored counter metadata in etcd: %s = %s", counterID, GetShardIds(counterShards))
	}

	return counterShards, nil
}

func GetShardIds(shards []*shardmetadata.Shard) []string {
	var shardsIds []string
	for _, shardMeta := range shards {
		shardsIds = append(shardsIds, shardMeta.ShardID)
	}
	return shardsIds

}

func GetShardObjList(shardIds []string) []*shardmetadata.Shard {
	var shardsList []*shardmetadata.Shard
	for _, shardID := range shardIds {
		shardData := new(shardmetadata.Shard)
		shardData.ShardID = shardID
		shardsList = append(shardsList, shardData)
	}
	return shardsList
}

// assignShards randomly selects shards for a counter.
func assignShards(shards []*shardmetadata.Shard) []*shardmetadata.Shard {
	// For simplicity, assign all shards (or select a random subset if needed).
	return shards
}
