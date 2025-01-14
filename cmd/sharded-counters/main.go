package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"sharded-counters/internal/etcd"
	"sharded-counters/internal/metrics"
	"sharded-counters/internal/server"
)

func main() {
	// Initialize etcd client
	cli, err := etcd.InitializeClient()
	if err != nil {
		log.Fatalf("Failed to initialize etcd client: %v", err)
	}
	defer cli.Close()

	// Get shard ID
	shardID := os.Getenv("POD_IP")
	if shardID == "" {
		shardID = "unknown"
	}

	// Start storing metrics in etcd
	go metrics.StoreMetrics(cli, shardID, 5*time.Second)

	// Setup health API
	http.HandleFunc("/health", server.HealthHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
