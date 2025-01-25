package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	countermetadata "sharded-counters/internal/counter_metadata"
	"sharded-counters/internal/etcd"
	"sharded-counters/internal/loadbalancer"
	"sharded-counters/internal/middleware"
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

type ShardCounterResponse struct {
	CounterID string `json:"counter_id"`
	Value     int64  `json:"value"`
}

const shardIncrementUrl = "counter/shard/increment"
const shardDecrementUrl = "counter/shard/decrement"

// CreateCounterHandler handles the counter creation API.
func CreateCounterHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve dependencies from context.
	deps, err := middleware.GetDependenciesFromContext(r.Context())
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve dependencies", err.Error())
		return
	}

	etcdManager := deps.EtcdManager

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

	shardIds, err := countermetadata.LoadOrStore(etcdManager, counterID)
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve CounterID", err.Error())
		return
	}

	// Respond with the assigned shards.
	resp := CounterResponse{
		CounterID: counterID,
		Shards:    countermetadata.GetShardIds(shardIds),
	}
	responsehandler.SendSuccessResponse(w, "Counter created successfully", resp)
}

// IncrementCounterHandler handles the counter increment API.
func IncrementCounterHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve dependencies from context.
	deps, err := middleware.GetDependenciesFromContext(r.Context())
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve dependencies", err.Error())
		return
	}

	etcdManager := deps.EtcdManager

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

	counterShards, err := countermetadata.LoadOrStore(etcdManager, req.CounterID)
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve CounterID", err.Error())
		return
	}

	// Load balancing logic
	strategy := &loadbalancer.MetricsStrategy{}
	lb := loadbalancer.NewLoadBalancer(counterShards, strategy, etcdManager)

	// Marshal the request payload.
	payload, err := json.Marshal(req)
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to marshal request payload", err.Error())
		return
	}
	// Forward the request to the selected shard.
	if err := lb.ForwardRequest("PUT", shardIncrementUrl, payload, nil); err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to forward request through load balancer", err.Error())
		return
	}

	responsehandler.SendSuccessResponse(w, "Counter incremented successfully", nil)
}

func DecrementCounterHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve dependencies from context.
	deps, err := middleware.GetDependenciesFromContext(r.Context())
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve dependencies", err.Error())
		return
	}

	etcdManager := deps.EtcdManager

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
	counterShards, metadataErr := countermetadata.GetCounterMetadata(etcdManager, req.CounterID)
	if etcd.IsKeyNotFound(metadataErr) {
		responsehandler.SendErrorResponse(w, http.StatusBadRequest, "Counter ID does not exist", "invalid value in counter_id")
		return

	}

	// Load balancing logic
	strategy := &loadbalancer.MetricsStrategy{}
	lb := loadbalancer.NewLoadBalancer(counterShards, strategy, etcdManager)

	// Marshal the request payload.
	payload, err := json.Marshal(req)
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to marshal request payload", err.Error())
		return
	}
	// Forward the request to the selected shard.
	if err := lb.ForwardRequest("PUT", shardDecrementUrl, payload, nil); err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to forward request through load balancer", err.Error())
		return
	}

	responsehandler.SendSuccessResponse(w, "Counter decremented successfully", nil)
}

func IncrementShardCounterHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve dependencies from context.
	deps, err := middleware.GetDependenciesFromContext(r.Context())
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve dependencies", err.Error())
		return
	}
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
	// call shard store to increment in memory shard counter (upsert behaviour)
	newValue := deps.CounterManager.Increment(req.CounterID)
	resp := ShardCounterResponse{
		CounterID: req.CounterID,
		Value:     newValue,
	}
	responsehandler.SendSuccessResponse(w, "Counter incremented successfully", resp)

}

func DecrementShardCounterHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve dependencies from context.
	deps, err := middleware.GetDependenciesFromContext(r.Context())
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve dependencies", err.Error())
		return
	}
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
	// call shard store to decrement in memory shard counter (upsert behaviour)
	newValue := deps.CounterManager.Decrement(req.CounterID)
	resp := ShardCounterResponse{
		CounterID: req.CounterID,
		Value:     newValue,
	}
	responsehandler.SendSuccessResponse(w, "Counter decremented successfully", resp)

}

func GetCounterHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve dependencies from context.
	deps, err := middleware.GetDependenciesFromContext(r.Context())
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve dependencies", err.Error())
		return
	}

	etcdManager := deps.EtcdManager

	// Retrieve `counter_id` from query parameters.
	counterID := r.URL.Query().Get("counter_id")
	if counterID == "" {
		responsehandler.SendErrorResponse(w, http.StatusBadRequest, "Counter ID is required", "Missing query parameter: counter_id")
		return
	}

	// Retrieve assigned shards (pods) for counter
	counterShards, metadataErr := countermetadata.GetCounterMetadata(etcdManager, counterID)
	if etcd.IsKeyNotFound(metadataErr) {
		responsehandler.SendErrorResponse(w, http.StatusBadRequest, "Counter ID does not exist", "invalid value in counter_id")
		return

	}

	// Aggregate sum of counter values by querying each shard.
	totalVal, err := aggregateCounterSum(counterID, counterShards, etcdManager)
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to aggregate sum", err.Error())
		return
	}
	resp := ShardCounterResponse{
		CounterID: counterID,
		Value:     totalVal,
	}

	responsehandler.SendSuccessResponse(w, "Counter aggregated successfully", resp)
}

func GetShardCounterHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve dependencies from context.
	deps, err := middleware.GetDependenciesFromContext(r.Context())
	if err != nil {
		responsehandler.SendErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve dependencies", err.Error())
		return
	}
	// Retrieve `counter_id` from query parameters.
	counterID := r.URL.Query().Get("counter_id")
	if counterID == "" {
		responsehandler.SendErrorResponse(w, http.StatusBadRequest, "Counter ID is required", "Missing query parameter: counter_id")
		return
	}
	// call shard store to get in memory  counter value.
	newValue := deps.CounterManager.Get(counterID)
	resp := ShardCounterResponse{
		CounterID: counterID,
		Value:     newValue,
	}
	responsehandler.SendSuccessResponse(w, "Counter Value fetched successfully", resp)

}

func aggregateCounterSum(counterID string, counterShards []*shardmetadata.Shard, etcdManager *etcd.EtcdManager) (int64, error) {
	lb := loadbalancer.NewLoadBalancer(counterShards, nil, etcdManager)
	lb.FilterHealthyShards()
	var total int64

	for _, shardData := range lb.GetShards() {
		// Send api request to each shard
		qyeryParams := map[string]string{"counter_id": counterID}
		respBody, statusCode, err := lb.ForwardRequestToShard("GET", shardData, "counter/shard", nil, qyeryParams)
		// read the response body {"success":true,"message":"","data":{"counter_id":"12345abcdef6ii978","value":1}}
		// Extract value and add it to total.
		if err != nil {
			return 0, fmt.Errorf("failed to query shard %s: %v (status Code: %d)", shardData.ShardID, err, statusCode)

		}
		response := &responsehandler.Response{}
		if err := json.Unmarshal([]byte(respBody), response); err != nil {
			return 0, fmt.Errorf("failed to parse response from shard %s: %v", shardData.ShardID, err)
		}
		// Check for API-level success.
		if !response.Success {
			return 0, fmt.Errorf("shard %s returned unsuccessful response for counter id %s", shardData.ShardID, counterID)
		}

		// Add the shard's counter value to the total.
		// Cast response.Data to the expected structure.
		dataMap, ok := response.Data.(map[string]interface{})
		if !ok {
			return 0, fmt.Errorf("invalid data format from shard %s", shardData.ShardID)
		}

		// Extract the counter value.
		value, ok := dataMap["value"].(float64) // JSON numbers are unmarshaled as float64.
		if !ok {
			return 0, fmt.Errorf("invalid data format from shard %s", shardData.ShardID)
		}
		// Extract the counter value.
		total += int64(value)

	}

	return total, nil

}
