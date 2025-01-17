package server

import (
	"encoding/json"
	"log"
	"net/http"
	countermetadata "sharded-counters/internal/counter_metadata"
	"sharded-counters/internal/etcd"
	"sharded-counters/internal/loadbalancer"
	"sharded-counters/internal/responsehandler"
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responsehandler.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate input.
	if req.Name == "" {
		responsehandler.SendErrorResponse(w, http.StatusBadRequest, "Counter name is required", "Missing field: name")
		return
	}

	// Generate a unique Counter ID.
	counterID, err := utils.GenerateUniqueID()
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to generate counter ID", err.Error())
		return
	}

	// Retrieve available shards (pods) from Etcd.
	shards, err := shardmetadata.GetAliveShards()
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve shards", err.Error())
		return
	}

	// Assign shards to the counter (randomly).
	assignedShards := assignShards(shards)

	// Save metadata to Etcd.
	if err := countermetadata.SaveCounterMetadata(counterID, assignedShards); err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to save metadata", err.Error())
		return
	}

	shardIds := countermetadata.GetShardIds(assignedShards)
	log.Printf("Stored counter metadata in etcd: %s = %s", counterID, shardIds)

	// Respond with the assigned shards.
	resp := CounterResponse{
		CounterID: counterID,
		Shards:    shardIds,
	}
	responsehandler.SendSuccessResponse(w, "Counter created successfully", resp)
}

// IncrementCounterHandler handles the counter increment API.
func IncrementCounterHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the request body.
	var req IncrementCounterReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responsehandler.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate input.
	if req.CounterID == "" {
		responsehandler.SendErrorResponse(w, http.StatusBadRequest, "Counter ID is required", "Missing field: counter_id")
		return
	}

	// Retrieve assigned shards (pods) for counter
	counterShards, metadataErr := countermetadata.GetCounterMetadata(req.CounterID)
	if etcd.IsKeyNotFound(metadataErr) {
		// Retrieve all available shards (pods) from Etcd.
		allAliveShards, err := shardmetadata.GetAliveShards()
		if err != nil {
			responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch shards metadata", err.Error())
			return
		}

		// Assign shards to the counter (randomly).
		counterShards = assignShards(allAliveShards)

		// Save metadata to Etcd.
		if err := countermetadata.SaveCounterMetadata(req.CounterID, counterShards); err != nil {
			responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to save metadata", err.Error())
			return
		}
		log.Printf("Stored counter metadata in etcd: %s = %s", req.CounterID, countermetadata.GetShardIds(counterShards))
	}

	// Load balancing logic
	strategy := &loadbalancer.MetricsStrategy{}
	lb := loadbalancer.NewLoadBalancer(counterShards, strategy)

	// Marshal the request payload.
	payload, err := json.Marshal(req)
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to marshal request payload", err.Error())
		return
	}

	// Forward the request to the selected shard.
	if err := lb.ForwardRequest(payload); err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to forward request through load balancer", err.Error())
		return
	}

	responsehandler.SendSuccessResponse(w, "Counter incremented successfully", nil)
}

// assignShards randomly selects shards for a counter.
func assignShards(shards []*shardmetadata.Shard) []*shardmetadata.Shard {
	// For simplicity, assign all shards (or select a random subset if needed).
	return shards
}
