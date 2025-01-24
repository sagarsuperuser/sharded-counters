package shardmetadata

import (
	"encoding/json"
	"fmt"
	"log"
	"sharded-counters/internal/etcd"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

const shardPrefix = "shards"

type Shard struct {
	ShardID        string  `json:"shard_id"`
	CPUUtilization float64 `json:"cpu_utilization"`
	Health         string  `json:"health"`
	UpdatedTime    string  `json:"updated_time"`
}

func fetchAndStoreMetrics(manager *etcd.EtcdManager, shardID string) error {
	utilizations, err := cpu.Percent(0, false)
	if err != nil {
		return fmt.Errorf("error fetching CPU utilization: %w", err)
	}

	utilization := 0.0
	if len(utilizations) > 0 {
		utilization = utilizations[0]
	}

	metrics := Shard{
		ShardID:        shardID,
		CPUUtilization: utilization,
		Health:         "ok",
		UpdatedTime:    time.Now().Format(time.RFC3339),
	}

	key := fmt.Sprintf("%s/%s", shardPrefix, shardID) // Overwrite previous value for the shard
	value, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("error marshaling metrics: %w", err)
	}

	err = manager.SaveMetadataWithLease(key, string(value), 6*time.Second)
	if err != nil {
		return fmt.Errorf("error storing metrics in etcd: %w", err)
	}

	log.Printf("Stored metrics in etcd with TTL: %s = %s", key, value)
	return nil
}

func StoreMetrics(manager *etcd.EtcdManager, shardID string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := fetchAndStoreMetrics(manager, shardID); err != nil {
				log.Printf("Error during metrics storage: %v", err)
			}
		}
	}
}

// GetAliveShards retrieves all shard keys from etcd and returns list of shard objects.
func GetAliveShards(manager etcd.Manager) ([]*Shard, error) {
	// Retrieve key values with the specified prefix from etcd
	keys, err := manager.GetKeysWithPrefix(shardPrefix)
	if err != nil {
		return nil, fmt.Errorf("error fetching shard keys from etcd: %w", err)
	}
	var healthyShards []*Shard
	prefix := fmt.Sprintf("%s/", shardPrefix)
	for _, shardKey := range keys {
		// shardKey => shards/<shard-id>
		if strings.HasPrefix(shardKey, prefix) {
			shardID := strings.TrimPrefix(shardKey, prefix)
			shardData := new(Shard)
			shardData.ShardID = shardID
			healthyShards = append(healthyShards, shardData)

		}
	}

	return healthyShards, nil
}

func GetShardMetrics(manager *etcd.EtcdManager, shardID string) (*Shard, error) {
	// Construct the key for the shard in etcd.
	key := fmt.Sprintf("%s/%s", shardPrefix, shardID)
	// Retrieve shard metadata from etcd.
	value, err := manager.Get(key)
	if err != nil {
		return nil, err
	}
	// Parse the shard metadata.
	shardMetadata := new(Shard)
	err = json.Unmarshal([]byte(value), shardMetadata)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling shard metadata for %s: %v", shardID, err)
	}
	return shardMetadata, nil
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
