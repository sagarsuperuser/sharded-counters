package etcd

import (
	"context"
	"errors"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Manager interface {
	Get(key string) (string, error)
	SaveMetadata(key, value string) error
	GetKeysWithPrefix(prefix string) ([]string, error)
}

// EtcdManager manages interactions with the Etcd client.
type EtcdManager struct {
	client *clientv3.Client
}

// NewEtcdManager initializes and returns a new EtcdManager instance.
func NewEtcdManager(endpoints []string, dialTimeout time.Duration) (*EtcdManager, error) {
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no etcd endpoints provided")
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &EtcdManager{client: client}, nil
}

// Close closes the Etcd client.
func (e *EtcdManager) Close() {
	if e.client != nil {
		e.client.Close()
	}
}

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

// SaveMetadata saves a key-value pair in Etcd.
func (e *EtcdManager) SaveMetadata(key, value string) error {
	if e.client == nil {
		return fmt.Errorf("etcd client is not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := e.client.Put(ctx, key, value)
	return err
}

// SaveMetadataWithLease saves a key-value pair in Etcd with a TTL.
func (e *EtcdManager) SaveMetadataWithLease(key, value string, ttl time.Duration) error {
	if e.client == nil {
		return fmt.Errorf("etcd client is not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	leaseResp, err := e.client.Grant(ctx, int64(ttl.Seconds()))
	if err != nil {
		return fmt.Errorf("error creating lease: %w", err)
	}

	_, err = e.client.Put(ctx, key, value, clientv3.WithLease(leaseResp.ID))
	return err
}

// GetKeysWithPrefix retrieves all keys matching a prefix from Etcd.
func (e *EtcdManager) GetKeysWithPrefix(prefix string) ([]string, error) {
	if e.client == nil {
		return nil, fmt.Errorf("etcd client is not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := e.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, kv := range resp.Kvs {
		keys = append(keys, string(kv.Key))
	}

	return keys, nil
}

// Get retrieves the value for a key from Etcd.
func (e *EtcdManager) Get(key string) (string, error) {
	if e.client == nil {
		return "", fmt.Errorf("etcd client is not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := e.client.Get(ctx, key)
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) == 0 {
		return "", &KeyNotFoundError{Key: key} // Return a specific error for "key not found"
	}

	return string(resp.Kvs[0].Value), nil
}
