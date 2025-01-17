package etcd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	client     *clientv3.Client
	clientOnce sync.Once
)

// KeyNotFoundError represents an error when a key is not found in etcd.
type KeyNotFoundError struct {
	Key string
}

func (e *KeyNotFoundError) Error() string {
	return fmt.Sprintf("key '%s' not found", e.Key)
}

// IsKeyNotFound checks if an error is of type KeyNotFoundError.
func IsKeyNotFound(err error) bool {
	var keyErr *KeyNotFoundError
	return errors.As(err, &keyErr)
}

// InitializeClient initializes a connection to etcd using the singleton pattern.
func InitializeClient() error {
	var err error
	clientOnce.Do(func() {
		etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
		if etcdEndpoints == "" {
			etcdEndpoints = "localhost:2379"
		}

		client, err = clientv3.New(clientv3.Config{
			Endpoints:   []string{etcdEndpoints},
			DialTimeout: 5 * time.Second,
		})
	})
	return err
}

// CloseEtcdClient closes the Etcd client.
func CloseEtcdClient() {
	if client != nil {
		client.Close()
	}
}

// GetClient returns the singleton Etcd client instance.
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

// GetWithPrefix fetches all keys from etcd with the specified prefix.
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

// Get fetches a key's value from etcd.
func Get(key string) (string, error) {
	resp, err := client.Get(context.Background(), key)
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) == 0 {
		return "", &KeyNotFoundError{Key: key} // Return a specific error for "key not found"
	}

	return string(resp.Kvs[0].Value), nil
}
