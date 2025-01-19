package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sharded-counters/internal/etcd"
	"sharded-counters/internal/middleware"
	"sharded-counters/internal/server"
	shardmetadata "sharded-counters/internal/shard_metadata"
	counter "sharded-counters/internal/shard_store"
	"time"

	"github.com/gorilla/mux"
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
	r := mux.NewRouter()

	// Define routes and enforce HTTP methods.
	r.Handle("/health", middleware.Middleware(deps, http.HandlerFunc(server.HealthHandler))).Methods(http.MethodGet)
	r.Handle("/counter/test", middleware.Middleware(deps, http.HandlerFunc(server.CreateCounterHandler))).Methods(http.MethodPost)
	r.Handle("/counter/increment", middleware.Middleware(deps, http.HandlerFunc(server.IncrementCounterHandler))).Methods(http.MethodPut)
	r.Handle("/counter/shard/increment", middleware.Middleware(deps, http.HandlerFunc(server.IncrementShardCounterHandler))).Methods(http.MethodPut)
	r.Handle("/counter/decrement", middleware.Middleware(deps, http.HandlerFunc(server.DecrementCounterHandler))).Methods(http.MethodPut)
	r.Handle("/counter/shard/decrement", middleware.Middleware(deps, http.HandlerFunc(server.DecrementShardCounterHandler))).Methods(http.MethodPut)

	r.Handle("/counter", middleware.Middleware(deps, http.HandlerFunc(server.GetCounterHandler))).Methods(http.MethodGet)
	r.Handle("/counter/shard", middleware.Middleware(deps, http.HandlerFunc(server.GetShardCounterHandler))).Methods(http.MethodGet)

	// Wrap the router with the middleware.
	http.Handle("/", r)
}
