package countermetadata_test

import (
	countermetadata "sharded-counters/internal/counter_metadata"
	"sharded-counters/internal/etcd"
	shardmetadata "sharded-counters/internal/shard_metadata"
	"testing"
	"time"
)

// MockEtcdManager implements the etcd.Manager interface for testing
type MockEtcdManager struct {
	store map[string]string
}

func NewMockEtcdManager() *MockEtcdManager {
	return &MockEtcdManager{store: make(map[string]string)}
}

func (m *MockEtcdManager) Get(key string) (string, error) {
	val, exists := m.store[key]
	if !exists {
		return "", &etcd.KeyNotFoundError{Key: key}
	}
	return val, nil
}

func (m *MockEtcdManager) SaveMetadata(key, value string) error {
	m.store[key] = value
	return nil
}

func (m *MockEtcdManager) GetKeysWithPrefix(prefix string) ([]string, error) {
	var keys []string
	for k := range m.store {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (m *MockEtcdManager) SaveMetadataWithLease(key, value string, ttl time.Duration) error {
	m.store[key] = value
	return nil
}

// Mock GetAliveShards function
// func MockGetAliveShards(manager etcd.Manager) ([]*shardmetadata.Shard, error) {
// 	return []*shardmetadata.Shard{
// 		{ShardID: "shard1"},
// 		{ShardID: "shard2"},
// 	}, nil
// }

func TestLoadOrStore(t *testing.T) {
	// Create a mock EtcdManager
	mockEtcd := NewMockEtcdManager()

	// Store shard metrics
	shardmetadata.FetchAndStoreMetrics(mockEtcd, "shard1")
	shardmetadata.FetchAndStoreMetrics(mockEtcd, "shard2")

	// Test Case 1: No existing metadata
	t.Run("No Existing Metadata", func(t *testing.T) {
		counterID := "test-counter"

		shards, err := countermetadata.LoadOrStore(mockEtcd, counterID)
		if err != nil {
			t.Fatalf("LoadOrStore failed: %v", err)
		}

		// Verify shards were assigned
		if len(shards) != 2 {
			t.Errorf("Expected 2 shards, got %d", len(shards))
		}

		// Verify metadata was saved
		savedShards, err := countermetadata.GetCounterMetadata(mockEtcd, counterID)
		if err != nil {
			t.Fatalf("GetCounterMetadata failed: %v", err)
		}
		if len(savedShards) != 2 {
			t.Errorf("Expected 2 saved shards, got %d", len(savedShards))
		}
	})

	// Test Case 2: Existing metadata
	t.Run("Existing Metadata", func(t *testing.T) {
		counterID := "test-counter"

		// Save initial counter metadata
		initialShards := []*shardmetadata.Shard{
			{ShardID: "shard1"},
			{ShardID: "shard2"},
		}
		err := countermetadata.SaveCounterMetadata(mockEtcd, counterID, initialShards)
		if err != nil {
			t.Fatalf("SaveCounterMetadata failed: %v", err)
		}
		shards, err := countermetadata.LoadOrStore(mockEtcd, counterID)
		if err != nil {
			t.Fatalf("LoadOrStore failed: %v", err)
		}

		// Verify that the existing metadata is retrieved
		if len(shards) != 2 || shards[0].ShardID != "shard1" {
			t.Errorf("Expected existing metadata to be retrieved, got %+v", shards)
		}
	})
}
