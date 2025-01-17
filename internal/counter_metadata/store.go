package countermetadata

import (
	"encoding/json"
	"fmt"
	"sharded-counters/internal/etcd"
	shardmetadata "sharded-counters/internal/shard_metadata"
)

const counterPrefix = "counters/" // Prefix used to identify counter keys in etcd

// saveCounterMetadata saves counter metadata in Etcd.
func SaveCounterMetadata(counterID string, shards []string) error {
	data, err := json.Marshal(shards)
	if err != nil {
		return fmt.Errorf("failed to marshal shards: %v", err)
	}
	key := fmt.Sprintf("%s/%s", counterPrefix, counterID)
	err = etcd.SaveMetadata(key, string(data))
	if err != nil {
		return fmt.Errorf("failed to store metadata in etcd: %v", err)
	}
	return nil
}

// getCounterMetadata retrieves counter metadata i.e. assigned shards from Etcd.
func GetCounterMetadata(counterID string) ([]*shardmetadata.Shard, error) {
	// Fetch the metadata from Etcd using the provided key
	key := fmt.Sprintf("%s/%s", counterPrefix, counterID)
	data, err := etcd.Get(key)
	if err != nil {
		return nil, err
	}
	// Unmarshal the JSON data into a slice of strings
	var shards []string
	err = json.Unmarshal([]byte(data), &shards)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %v", err)
	}

	var shardsList []*shardmetadata.Shard
	for _, shardID := range shards {
		shardData := new(shardmetadata.Shard)
		shardData.ShardID = shardID
		shardsList = append(shardsList, shardData)
	}

	return shardsList, nil

}
