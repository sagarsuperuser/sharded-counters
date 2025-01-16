package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sharded-counters/internal/etcd"
	"sharded-counters/internal/metrics"
	"sharded-counters/internal/utils"
	"strings"
)

const shardPrefix = "shards/" // Prefix used to identify shard keys in etcd

// CounterRequest represents the request payload for creating a counter.
type IncrementCounterReq struct {
	CounterID string `json:"counter_id"`
}

type CounterRequest struct {
	Name string `json:"name"`
}

// CounterResponse represents the response payload after creating a counter.
type CounterResponse struct {
	CounterID   string   `json:"counter_id"`
	CounterName string   `json:"counter_name"`
	Shards      []string `json:"shards"`
}

// CreateCounterHandler handles the counter creation API.
func CreateCounterHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the request body.
	var req CounterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input.
	if req.Name == "" {
		http.Error(w, "Counter name is required", http.StatusBadRequest)
		return
	}

	// Generate a unique Counter ID.
	counterID, err := utils.GenerateUniqueID()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate counter ID: %v", err), http.StatusInternalServerError)
		return
	}

	// Retrieve available shards (pods) from Etcd.
	shards, err := getShards()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve shards: %v", err), http.StatusInternalServerError)
		return
	}

	// Assign shards to the counter (randomly).
	assignedShards := assignShards(shards)

	// Save metadata to Etcd.
	err = saveCounterMetadata(counterID, assignedShards)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save metadata: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Stored counter metadata in etcd: %s = %s", counterID, assignedShards)

	// Respond with the assigned shards.
	resp := CounterResponse{
		CounterID: counterID,
		Shards:    assignedShards,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// IncrementCounterHandler handles the counter increment API.
func IncrementCounterHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the request body.
	var req IncrementCounterReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input.
	if req.CounterID == "" {
		http.Error(w, "Counter id is required", http.StatusBadRequest)
		return
	}

	// Retrieve assigned shards (pods) for counter
	counterShards, err := getCounterMetadata(req.CounterID)

	// Handle the "key not found" case
	if err == etcd.ErrKeyNotFound {
		// Retrieve available shards (pods) from Etcd.
		shards, err := getShards()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to retrieve shards: %v", err), http.StatusInternalServerError)
			return
		}

		// Assign shards to the counter (randomly).
		counterShards = assignShards(shards)

		// Save metadata to Etcd.
		err = saveCounterMetadata(req.CounterID, counterShards)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to save metadata: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("Stored counter metadata in etcd: %s = %s", req.CounterID, counterShards)

	}

	// Load balancing logic ->
	// Filter healthy shards stored in etcd with key shards/<shard-id> - Done
	// Select a shard based on least cpu utilization metric, and forward the request with req.CounterID to selected shard (using api call)

	// Filter healthy shards.
	healthyShards, err := filterHealthyShards(counterShards)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve healthy shards: %v", err), http.StatusInternalServerError)
		return
	}

	// Select the shard with the least CPU utilization.
	selectedShard, err := selectShardByCPU(healthyShards)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to select a shard: %v", err), http.StatusInternalServerError)
		return
	}

	// Forward the request to the selected shard.
	err = forwardRequestToShard(selectedShard, req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to forward request to shard: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Request forwarded to shard %s for counter %s", selectedShard.ShardID, req.CounterID)
	w.WriteHeader(http.StatusOK)
}

// GetShardsFromEtcd retrieves all shard keys from etcd and returns them as a slice of strings.
func getShards() ([]string, error) {
	// Retrieve keys with the specified prefix from etcd
	keys, err := etcd.GetKeysWithPrefix(shardPrefix)
	if err != nil {
		return nil, fmt.Errorf("error fetching shard keys from etcd: %w", err)
	}

	var shardIDs []string
	for _, key := range keys {
		// Extract the shard-id by trimming the prefix
		if strings.HasPrefix(key, shardPrefix) {
			shardID := strings.TrimPrefix(key, shardPrefix)
			shardIDs = append(shardIDs, shardID)
		}
	}

	return shardIDs, nil
}

// assignShards randomly selects shards for a counter.
func assignShards(shards []string) []string {
	// For simplicity, assign all shards (or select a random subset if needed).
	return shards
}

// saveCounterMetadata saves counter metadata in Etcd.
func saveCounterMetadata(counterID string, shards []string) error {
	data, err := json.Marshal(shards)
	if err != nil {
		return fmt.Errorf("failed to marshal shards: %v", err)
	}

	err = etcd.SaveMetadata(counterID, string(data))
	if err != nil {
		return fmt.Errorf("failed to store metadata in etcd: %v", err)
	}
	return nil
}

// getCounterMetadata retrieves counter metadata i.e. assigned shards from Etcd.
func getCounterMetadata(counterID string) ([]string, error) {
	// Fetch the metadata from Etcd using the provided key
	data, err := etcd.GetKey(counterID)
	if err != nil {
		return nil, err
	}
	// Unmarshal the JSON data into a slice of strings
	var shards []string
	err = json.Unmarshal([]byte(data), &shards)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %v", err)
	}

	return shards, nil

}

func filterHealthyShards(shardIDs []string) ([]metrics.Metrics, error) {
	var healthyShards []metrics.Metrics
	for _, shardID := range shardIDs {
		// Construct the key for the shard in etcd.
		key := fmt.Sprintf("%s%s", shardPrefix, shardID)

		// Retrieve shard metadata from etcd.
		data, err := etcd.GetKey(key)
		if err != nil {
			log.Printf("Error retrieving shard metadata for %s: %v", shardID, err)
			continue
		}

		// Parse the shard metadata.
		var shardMetadata metrics.Metrics
		err = json.Unmarshal([]byte(data), &shardMetadata)
		if err != nil {
			log.Printf("Error unmarshaling shard metadata for %s: %v", shardID, err)
			continue
		}

		// Check if the shard is healthy.
		if shardMetadata.Health == "ok" {
			healthyShards = append(healthyShards, shardMetadata)
		}
	}

	return healthyShards, nil
}

func selectShardByCPU(shards []metrics.Metrics) (metrics.Metrics, error) {
	var selectedShard metrics.Metrics
	var minCPU float64 = 100.0 // Start with a high CPU value.

	for _, shard := range shards {
		if shard.CPUUtilization < minCPU {
			minCPU = shard.CPUUtilization
			selectedShard = shard
		}
	}

	if minCPU == 100.0 {
		return metrics.Metrics{}, fmt.Errorf("no shard with valid CPU metrics found")
	}

	return selectedShard, nil
}

func forwardRequestToShard(shard metrics.Metrics, req IncrementCounterReq) error {
	// Construct the URL for the shard's API endpoint.
	shardURL := fmt.Sprintf("http://%s/counter/shard/increment", shard.ShardID)

	// Marshal the request payload.
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request payload: %v", err)
	}

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

// getShards retrieves available pod IPs using the headless service FQDN.
// func getShards() ([]string, error) {
// 	// Retrieve namespace and service name from environment variables.
// 	namespace := os.Getenv("SHARD_NAMESPACE")      // e.g., "default"
// 	serviceName := os.Getenv("SHARD_SERVICE_NAME") // e.g., "shards-headless"

// 	if namespace == "" || serviceName == "" {
// 		return nil, fmt.Errorf("environment variables SHARD_NAMESPACE or SHARD_SERVICE_NAME are not set")
// 	}

// 	// Construct the FQDN of the headless service.
// 	serviceFQDN := fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace)

// 	// Resolve the service FQDN to pod IPs.
// 	ips, err := net.LookupIP(serviceFQDN)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to resolve FQDN %s: %v", serviceFQDN, err)
// 	}

// 	// Collect IPs into a string slice.
// 	var shards []string
// 	for _, ip := range ips {
// 		shards = append(shards, ip.String())
// 	}

// 	if len(shards) == 0 {
// 		return nil, fmt.Errorf("no pods found for service %s", serviceFQDN)
// 	}

// 	return shards, nil
// }
