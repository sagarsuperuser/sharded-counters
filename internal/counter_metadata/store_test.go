package countermetadata_test

import (
	"encoding/json"
	"fmt"
	countermetadata "sharded-counters/internal/counter_metadata"
	"sharded-counters/internal/etcd"
	shardmetadata "sharded-counters/internal/shard_metadata"
	"testing"
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

// Mock GetAliveShards function
func MockGetAliveShards(manager etcd.Manager) ([]*shardmetadata.Shard, error) {
	return []*shardmetadata.Shard{
		{ShardID: "shard1"},
		{ShardID: "shard2"},
	}, nil
}

func TestLoadOrStore(t *testing.T) {
	// Create a mock EtcdManager
	mockEtcd := NewMockEtcdManager()

	// Inject the mock function for GetAliveShards
	getAliveShards := MockGetAliveShards

	// Test Case 1: No existing metadata
	t.Run("No Existing Metadata", func(t *testing.T) {
		counterID := "test-counter"

		shards, err := countermetadata.LoadOrStore(mockEtcd, counterID, getAliveShards)
		if err != nil {
			t.Fatalf("LoadOrStore failed: %v", err)
		}

		// Verify shards were assigned
		if len(shards) != 2 {
			t.Errorf("Expected 2 shards, got %d", len(shards))
		}

		// Verify metadata was saved
		key := fmt.Sprintf("%s/%s", countermetadata.CounterPrefix, counterID)
		savedData, _ := mockEtcd.Get(key)
		var savedShards []string
		_ = json.Unmarshal([]byte(savedData), &savedShards)
		if len(savedShards) != 2 {
			t.Errorf("Expected 2 saved shards, got %d", len(savedShards))
		}
	})

	// Test Case 2: Existing metadata
	t.Run("Existing Metadata", func(t *testing.T) {
		counterID := "test-counter"

		// Save initial metadata
		initialShards := []string{"shard1", "shard2"}
		data, _ := json.Marshal(initialShards)
		key := fmt.Sprintf("%s/%s", countermetadata.CounterPrefix, counterID)
		mockEtcd.SaveMetadata(key, string(data))

		shards, err := countermetadata.LoadOrStore(mockEtcd, counterID, getAliveShards)
		if err != nil {
			t.Fatalf("LoadOrStore failed: %v", err)
		}

		// Verify that the existing metadata is retrieved
		if len(shards) != 2 || shards[0].ShardID != "shard1" {
			t.Errorf("Expected existing metadata to be retrieved, got %+v", shards)
		}
	})
}
