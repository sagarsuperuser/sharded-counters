package loadbalancer

import (
	"fmt"
	"log"
	"net/http"
	shardmetadata "sharded-counters/internal/shard_metadata"
	"strings"
	"sync"
)

// const shardIncrApi = "counter/shard/increment"
const shardIncrApi = "health"

// LoadBalancer manages shard selection based on specific strategies.
type LoadBalancer struct {
	shards            []*shardmetadata.Shard
	selectionStrategy SelectionStrategy
}

var (
	instance *LoadBalancer
	once     sync.Once
)

// SelectionStrategy defines the interface for shard selection strategies.
type SelectionStrategy interface {
	SelectShard(shards []*shardmetadata.Shard) (*shardmetadata.Shard, error)
}

// InitializeLoadBalancer initializes the singleton LoadBalancer instance with the provided shards and strategy.
// It must be called before GetInstance.
func InitializeLoadBalancer(shards []*shardmetadata.Shard, strategy SelectionStrategy) {
	once.Do(func() {
		instance = &LoadBalancer{
			shards:            shards,
			selectionStrategy: strategy,
		}
	})
}

// GetInstance returns the singleton instance of LoadBalancer.
// Ensure InitializeLoadBalancer is called first.
func GetInstance() *LoadBalancer {
	if instance == nil {
		log.Fatalf("LoadBalancer instance is not initialized. Call InitializeLoadBalancer first.")
	}
	return instance
}

func (lb *LoadBalancer) ForwardRequest(payload []byte) error {
	// Select the shard based on selection stratedy
	selectedShard, err := lb.selectionStrategy.SelectShard(lb.shards)
	if err != nil {
		return fmt.Errorf("failed to select a shard: %v", err)
	}

	// Forward the request to the selected shard.
	err = lb.forwardRequestToShard(selectedShard, payload)
	if err != nil {
		return fmt.Errorf("failed to forward request to shard: %v", err)
	}

	log.Printf("Request forwarded to shard %s with payload %s", selectedShard.ShardID, payload)
	return nil

}

func (lb *LoadBalancer) forwardRequestToShard(shard *shardmetadata.Shard, payload []byte) error {
	// Construct the URL for the shard's API endpoint.
	shardURL := fmt.Sprintf("http://%s/%s", shard.ShardID, shardIncrApi)

	// Send the request to the shard.
	resp, err := http.Post(shardURL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("failed to forward request to shard: %v", err)
	}
	defer resp.Body.Close()

	// Check for non-2xx status codes.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("shard returned error status: %d", resp.StatusCode)
	}

	return nil
}
