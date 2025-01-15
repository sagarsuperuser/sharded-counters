package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sharded-counters/internal/etcd"
	"sharded-counters/internal/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// CounterRequest represents the request payload for creating a counter.
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

	// Retrieve available shards (pods) from Kubernetes.
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
		CounterID:   counterID,
		CounterName: req.Name,
		Shards:      assignedShards,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// getShards retrieves available pods from a Kubernetes headless service.
func getShards() ([]string, error) {
	// Create Kubernetes client.
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	// List pods in the headless service.
	namespace := os.Getenv("SHARD_NAMESPACE") // Retrieve namespace from environment variable.
	serviceName := os.Getenv("SHARD_SERVICE_NAME")

	if namespace == "" || serviceName == "" {
		return nil, fmt.Errorf("environment variables SHARD_NAMESPACE or SHARD_SERVICE_NAME are not set")
	}

	// Construct label selector based on the service's app label.
	labelSelector := fmt.Sprintf("app=%s", serviceName)

	podClient := clientset.CoreV1().Pods(namespace)
	podList, err := podClient.List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}

	// Collect pod IPs.
	var shards []string
	for _, pod := range podList.Items {
		shards = append(shards, pod.Status.PodIP)
	}
	return shards, nil
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
