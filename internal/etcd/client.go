package etcd

import (
	"os"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// InitializeClient initializes a connection to etcd
func InitializeClient() (*clientv3.Client, error) {
	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		etcdEndpoints = "localhost:2379"
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdEndpoints},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return cli, nil
}
