package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/shirou/gopsutil/cpu"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Metrics struct {
	ShardID        string  `json:"shard_id"`
	CPUUtilization float64 `json:"cpu_utilization"`
	Health         string  `json:"health"`
	UpdatedTime    string  `json:"updated_time"`
}

func fetchAndStoreMetrics(cli *clientv3.Client, shardID string) error {
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

	// Create a lease with a TTL of 6 seconds
	leaseResp, err := cli.Grant(context.Background(), 6)
	if err != nil {
		return fmt.Errorf("error creating lease: %w", err)
	}

	// Put the key-value pair with the lease
	_, err = cli.Put(context.Background(), key, string(value), clientv3.WithLease(leaseResp.ID))
	if err != nil {
		return fmt.Errorf("error storing metrics in etcd: %w", err)
	}

	log.Printf("Stored metrics in etcd with TTL: %s = %s", key, value)
	return nil
}

func StoreMetrics(cli *clientv3.Client, shardID string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := fetchAndStoreMetrics(cli, shardID); err != nil {
				log.Printf("Error during metrics storage: %v", err)
			}
		}
	}
}
