package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	countermetadata "sharded-counters/internal/counter_metadata"
	"sharded-counters/internal/etcd"
	"sharded-counters/internal/loadbalancer"
	shardmetadata "sharded-counters/internal/shard_metadata"
	"sharded-counters/internal/utils"
)

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
	shards, err := shardmetadata.GetShards()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve shards: %v", err), http.StatusInternalServerError)
		return
	}

	// Assign shards to the counter (randomly).
	assignedShards := assignShards(shards)

	// Save metadata to Etcd.
	err = countermetadata.SaveCounterMetadata(counterID, assignedShards)
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

	// Retrieve all available shards (pods) from Etcd.
	shards, err := shardmetadata.GetShards()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch shards metadata: %v", err), http.StatusInternalServerError)
		return
	}
	// Retrieve assigned shards (pods) for counter
	counterShards, metadataErr := countermetadata.GetCounterMetadata(req.CounterID)
	// Handle the "key not found" case
	if metadataErr == etcd.ErrKeyNotFound {

		// Assign shards to the counter (randomly).
		counterShards = assignShards(shards)

		// Save metadata to Etcd.
		err = countermetadata.SaveCounterMetadata(req.CounterID, counterShards)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to save metadata: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("Stored counter metadata in etcd: %s = %s", req.CounterID, counterShards)

	}

	// Load balancing logic ->
	// Filter healthy shards stored in etcd with key shards/<shard-id> - Done
	// Select a shard based on least cpu utilization metric, and forward the request with req.CounterID to selected shard (using api call)
	strategy := &loadbalancer.MetricsStrategy{}
	loadbalancer.InitializeLoadBalancer(shards, strategy)
	// Get the LoadBalancer instance.
	lb := loadbalancer.GetInstance()

	// Marshal the request payload.
	payload, err := json.Marshal(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal request payload: %v", err), http.StatusInternalServerError)
	}
	err = lb.ForwardRequest(payload)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed forwading request through loadbalancer: %v", err), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

// assignShards randomly selects shards for a counter.
func assignShards(shards []*shardmetadata.Shard) []string {
	// For simplicity, assign all shards (or select a random subset if needed).
	var newShards []string
	for _, shard := range shards {
		newShards = append(newShards, shard.ShardID)
	}
	return newShards
}
