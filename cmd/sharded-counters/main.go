package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"sharded-counters/internal/etcd"
	"sharded-counters/internal/middleware"
	"sharded-counters/internal/server"
	shardmetadata "sharded-counters/internal/shard_metadata"
	counter "sharded-counters/internal/shard_store"
)

func main() {
	// Initialize CounterManager and EtcdManager.
	counterManager := counter.GetCounterManager()
	// Initialize etcd client
	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		etcdEndpoints = "localhost:2379"
	}
	etcdManager, err := etcd.NewEtcdManager([]string{etcdEndpoints}, 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to initialize etcd client: %v", err)
	}
	defer etcdManager.Close()

	// Get shard ID
	shardID := os.Getenv("POD_IP")
	if shardID == "" {
		shardID = "unknown"
	}

	// shard periodic job interval in seconds
	shardInterval := 5 * time.Second

	// Start storing metrics in etcd
	servType := os.Getenv("SERVICE_TYPE")
	if servType == "" {
		servType = "app"
	}

	if servType == "shard" {
		go shardmetadata.StoreMetrics(etcdManager, shardID, shardInterval)

	}

	// Create a Dependencies container.
	deps := &middleware.Dependencies{
		CounterManager: counterManager,
		EtcdManager:    etcdManager,
	}

	startAPI(deps)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func startAPI(deps *middleware.Dependencies) {
	// Setup health API
	http.Handle("/health", middleware.Middleware(deps, http.HandlerFunc(server.HealthHandler)))
	http.Handle("/counter/test", middleware.Middleware(deps, http.HandlerFunc(server.CreateCounterHandler)))
	http.Handle("/counter/increment", middleware.Middleware(deps, http.HandlerFunc(server.IncrementCounterHandler)))
	http.Handle("/counter/shard/increment", middleware.Middleware(deps, http.HandlerFunc(server.IncrementShardCounterHandler)))

}
