package metrics

import (
	"encoding/json"
	"fmt"
	"log"
	"sharded-counters/internal/etcd"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

type Metrics struct {
	ShardID        string  `json:"shard_id"`
	CPUUtilization float64 `json:"cpu_utilization"`
	Health         string  `json:"health"`
	UpdatedTime    string  `json:"updated_time"`
}

func fetchAndStoreMetrics(shardID string) error {
	utilizations, err := cpu.Percent(0, false)
	if err != nil {
		return fmt.Errorf("error fetching CPU utilization: %w", err)
	}

	utilization := 0.0
	if len(utilizations) > 0 {
		utilization = utilizations[0]
	}

	metrics := Metrics{
		ShardID:        shardID,
		CPUUtilization: utilization,
		Health:         "ok",
		UpdatedTime:    time.Now().Format(time.RFC3339),
	}

	key := fmt.Sprintf("metrics/%s", shardID) // Overwrite previous value for the shard
	value, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("error marshaling metrics: %w", err)
	}

	err = etcd.SaveMetadataWithLease(key, string(value), 6)
	if err != nil {
		return fmt.Errorf("error storing metrics in etcd: %w", err)
	}

	log.Printf("Stored metrics in etcd with TTL: %s = %s", key, value)
	return nil
}

func StoreMetrics(shardID string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := fetchAndStoreMetrics(shardID); err != nil {
				log.Printf("Error during metrics storage: %v", err)
			}
		}
	}
}
