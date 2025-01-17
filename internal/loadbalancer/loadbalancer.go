package loadbalancer

import (
	"fmt"
	"log"
	"net/http"
	shardmetadata "sharded-counters/internal/shard_metadata"
	"strings"
)

// const shardIncrApi = "counter/shard/increment"
const shardIncrApi = "health"
const shardPort = "8080"

// LoadBalancer manages shard selection based on specific strategies.
type LoadBalancer struct {
	shards            []*shardmetadata.Shard
	selectionStrategy SelectionStrategy
}

// SelectionStrategy defines the interface for shard selection strategies.
type SelectionStrategy interface {
	SelectShard(shards []*shardmetadata.Shard) (*shardmetadata.Shard, error)
}

// NewLoadBalancer creates and initializes a new LoadBalancer instance.
func NewLoadBalancer(shards []*shardmetadata.Shard, strategy SelectionStrategy) *LoadBalancer {
	return &LoadBalancer{
		shards:            shards,
		selectionStrategy: strategy,
	}
}

func (lb *LoadBalancer) SetShards(shards []*shardmetadata.Shard) error {
	lb.shards = shards
	return nil
}

func (lb *LoadBalancer) GetShards() []*shardmetadata.Shard {
	return lb.shards
}

func (lb *LoadBalancer) ForwardRequest(payload []byte) error {
	// Filter out healthy shards and set new shards, key => shards/<shard-id>
	lb.filterHealthyShards()
	// Select the shard based on selection strategy
	selectedShard, err := lb.selectionStrategy.SelectShard(lb.GetShards())
	if err != nil {
		return fmt.Errorf("failed to select a shard: %v", err)
	}

	// Forward the request to the selected shard.
	err = lb.forwardRequestToShard(selectedShard, payload)
	if err != nil {
		return err
	}

	log.Printf("Request forwarded to shard %s with payload %s", selectedShard.ShardID, payload)
	return nil
}

func (lb *LoadBalancer) filterHealthyShards() error {
	var healthyShards []*shardmetadata.Shard
	for _, shardData := range lb.GetShards() {
		// fetch shard metrics from etcd
		shardMetrics, err := shardmetadata.GetShardMetrics(shardData.ShardID)
		if err != nil {
			log.Printf("error fetching shard metrics from etcd: %v", err)
			continue
		}

		// Check if the shard is healthy.
		if shardMetrics.Health == "ok" {
			healthyShards = append(healthyShards, shardMetrics)
		}
	}
	lb.SetShards(healthyShards)
	return nil
}

func (lb *LoadBalancer) forwardRequestToShard(shard *shardmetadata.Shard, payload []byte) error {
	// Construct the URL for the shard's API endpoint.
	shardURL := fmt.Sprintf("http://%s:%s/%s", shard.ShardID, shardPort, shardIncrApi)

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
