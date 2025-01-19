package loadbalancer

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sharded-counters/internal/etcd"
	shardmetadata "sharded-counters/internal/shard_metadata"
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

func (lb *LoadBalancer) ForwardRequest(method string, urlPath string, payload []byte, queryParams map[string]string) error {
	// Filter out healthy shards and set new shards, key => shards/<shard-id>
	lb.FilterHealthyShards()
	// Select the shard based on selection strategy
	selectedShard, err := lb.selectionStrategy.SelectShard(lb.GetShards())
	if err != nil {
		return fmt.Errorf("failed to select a shard: %v", err)
	}

	// Forward the request to the selected shard.
	_, _, err = lb.ForwardRequestToShard(method, selectedShard, urlPath, payload, queryParams)
	if err != nil {
		return err
	}
	return nil
}

func (lb *LoadBalancer) FilterHealthyShards() error {
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

func (lb *LoadBalancer) ForwardRequestToShard(method string, shard *shardmetadata.Shard, urlPath string, payload []byte, queryParams map[string]string) (string, int, error) {
	// Construct the base URL for the shard's API endpoint.
	baseURL := fmt.Sprintf("http://%s:%s/%s", shard.ShardID, shardPort, urlPath)

	// Parse the base URL to append query parameters.
	urlObj, err := url.Parse(baseURL)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse URL: %v", err)
	}

	// Add query parameters if provided.
	if queryParams != nil {
		q := urlObj.Query()
		for key, value := range queryParams {
			q.Add(key, value)
		}
		urlObj.RawQuery = q.Encode()
	}

	// Prepare the request body (nil for GET requests).
	var bodyReader io.Reader
	if method != http.MethodGet && len(payload) > 0 {
		bodyReader = strings.NewReader(string(payload))
	}

	// Create the HTTP request.
	req, err := http.NewRequest(method, urlObj.String(), bodyReader)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request for shard: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Log the request details.
	logRequest(req, payload)

	// Execute the request.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to forward request to shard: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body.
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read shard response body: %v", err)
		return "", resp.StatusCode, fmt.Errorf("failed to read response body: %v", err)
	}

	// Log the response details.
	logResponse(resp, responseBody)

	// Check for non-2xx status codes.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return string(responseBody), resp.StatusCode, fmt.Errorf("shard returned error status: %d", resp.StatusCode)
	}

	return string(responseBody), resp.StatusCode, nil
}

// logRequest logs the details of the outgoing HTTP request.
func logRequest(req *http.Request, payload []byte) {
	log.Printf("Request: Method=%s, URL=%s, Headers=%v, Payload=%s", req.Method, req.URL.String(), req.Header, string(payload))
}

// logResponse logs the response details from the shard API.
func logResponse(resp *http.Response, body []byte) {
	log.Printf("Response: Status Code=%d, Body=%s", resp.StatusCode, string(body))
}
