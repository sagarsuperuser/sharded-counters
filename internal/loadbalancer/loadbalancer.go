package loadbalancer

import (
	"fmt"
	"log"
	"net/http"
	"sharded-counters/internal/etcd"
	shardmetadata "sharded-counters/internal/shard_metadata"
	"sharded-counters/internal/utils"
	"strings"
)

const shardPort = "8080"

// LoadBalancer manages shard selection based on specific strategies.
type LoadBalancer struct {
	shards            []*shardmetadata.Shard
	selectionStrategy SelectionStrategy
	etcdClient        *etcd.EtcdManager
}

// SelectionStrategy defines the interface for shard selection strategies.
type SelectionStrategy interface {
	SelectShard(shards []*shardmetadata.Shard) (*shardmetadata.Shard, error)
}

// NewLoadBalancer creates and initializes a new LoadBalancer instance.
func NewLoadBalancer(shards []*shardmetadata.Shard, strategy SelectionStrategy, eClient *etcd.EtcdManager) *LoadBalancer {
	return &LoadBalancer{
		shards:            shards,
		selectionStrategy: strategy,
		etcdClient:        eClient,
	}
}

func (lb *LoadBalancer) SetShards(shards []*shardmetadata.Shard) error {
	lb.shards = shards
	return nil
}

func (lb *LoadBalancer) GetShards() []*shardmetadata.Shard {
	return lb.shards
}

func (lb *LoadBalancer) ForwardRequest(urlPath string, payload []byte) error {
	// Filter out healthy shards and set new shards, key => shards/<shard-id>
	lb.filterHealthyShards()
	// Select the shard based on selection strategy
	selectedShard, err := lb.selectionStrategy.SelectShard(lb.GetShards())
	if err != nil {
		return fmt.Errorf("failed to select a shard: %v", err)
	}

	// Forward the request to the selected shard.
	err = lb.forwardRequestToShard(selectedShard, urlPath, payload)
	if err != nil {
		return err
	}
	return nil
}

func (lb *LoadBalancer) filterHealthyShards() error {
	var healthyShards []*shardmetadata.Shard
	for _, shardData := range lb.GetShards() {
		// fetch shard metrics from etcd
		shardMetrics, err := shardmetadata.GetShardMetrics(lb.etcdClient, shardData.ShardID)
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

func (lb *LoadBalancer) forwardRequestToShard(shard *shardmetadata.Shard, urlPath string, payload []byte) error {
	// Construct the URL for the shard's API endpoint.
	shardURL := fmt.Sprintf("http://%s:%s/%s", shard.ShardID, shardPort, urlPath)

	// Send the request to the shard using PUT method.
	req, err := http.NewRequest(http.MethodPut, shardURL, strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("failed to create request for shard: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute the request.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to forward request to shard: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("Request forwarded to shard %s with payload %s", shard.ShardID, payload)
	utils.LogResponse(resp)

	// Check for non-2xx status codes.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("shard returned error status: %d", resp.StatusCode)
	}

	return nil
}
