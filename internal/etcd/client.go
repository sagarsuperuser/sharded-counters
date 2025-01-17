package etcd

import (
	"context"
	"fmt"
	"os"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var client *clientv3.Client

// ErrKeyNotFound is returned when the key does not exist in etcd.
var ErrKeyNotFound = fmt.Errorf("key not found")

// InitializeClient initializes a connection to etcd
func InitializeClient() error {
	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		etcdEndpoints = "localhost:2379"
	}

	var err error
	client, err = clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdEndpoints},
		DialTimeout: 5 * time.Second,
	})
	return err
}

// CloseEtcdClient closes the Etcd client.
func CloseEtcdClient() {
	if client != nil {
		client.Close()
	}
}

// GetClient returns the Etcd client instance.
func GetClient() *clientv3.Client {
	return client
}

// SaveCounterMetadata saves metadata in Etcd.
func SaveMetadata(key string, value string) error {
	_, err := client.Put(context.Background(), key, value)
	return err
}

func SaveMetadataWithLease(key string, value string, duration int64) error {
	// Create a lease with a TTL of 6 seconds
	leaseResp, err := client.Grant(context.Background(), 6)
	if err != nil {
		return fmt.Errorf("error creating lease: %w", err)
	}

	// Put the key-value pair with the lease
	_, err = client.Put(context.Background(), key, string(value), clientv3.WithLease(leaseResp.ID))

	return err
}

// GetWithPrefix fetches all keeys from etcd with the specified prefix.
func GetKeysWithPrefix(prefix string) ([]string, error) {
	resp, err := client.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, kv := range resp.Kvs {
		keys = append(keys, string(kv.Key))
	}

	return keys, nil
}

// GetKey fetches key value from etcd.
func Get(key string) (string, error) {
	resp, err := client.Get(context.Background(), key)
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) == 0 {
		return "", ErrKeyNotFound // Return a specific error for "key not found"
	}

	return string(resp.Kvs[0].Value), nil
}
