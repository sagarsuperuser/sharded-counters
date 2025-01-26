# Sharded Counters

![Sharded Counter System](images/sharded-counter.png)

## Overview

In distributed systems, maintaining accurate counters under high concurrency can be challenging due to bottlenecks and contention. Sharded counters address this by distributing the counter's value across multiple shards, allowing for scalable and efficient counting mechanisms.

## Features

- **High Throughput:** Distributes load across multiple shards to handle a large number of operations concurrently.
- **Fault Tolerance:** Ensures system reliability by mitigating single points of failure.
- **Scalability:** Dynamically adjusts the number of shards based on workload demands.

## Architecture

The system comprises the following components:

1. **App Server:** Processes API requests for counter operations.
2. **Load Balancer:** Distributes requests across shards based on a metrics-driven strategy (e.g., CPU utilization) to optimize resource usage.
3. **Service Registry (e.g., Etcd):** Maintains metadata about active shards and their counter-to-shard mappings.
4. **Shard Cluster:** Managed by Kubernetes, each pod handles a portion of the counter's data.
5. **Persistent Storage (e.g., Cassandra):** Stores counter data for low-latency read operations and durability.

## Workflow

1. **Counter Creation:** A new counter is initialized and distributed across a predefined number of shards.
2. **Increment/Decrement Operations:** Incoming requests are routed to the Load Balancer, which identifies the optimal shard for the operation based on metrics such as CPU utilization and forwards the request accordingly.
3. **Read Operations:** Appropriate shards are queried to retrieve values. The results are combined to compute the total counter value.

## Getting Started

### Prerequisites

- [Go](https://golang.org/doc/install) (version 1.16 or higher)
- [Docker](https://docs.docker.com/get-docker/)
- [Kubernetes](https://kubernetes.io/docs/setup/) cluster
- [Etcd](https://etcd.io/docs/v3.4.0/getting-started/) service
- [Cassandra](https://cassandra.apache.org/_/quickstart.html) database

### Installation

1. **Clone the Repository:**

   ```bash
   git clone https://github.com/sagarsuperuser/sharded-counters.git
   cd sharded-counters
   ```

2. **Run the Application:**

   ```bash
   go run ./cmd/sharded-counters
   ```

3. **Build Application using Docker:**

   ```bash
   docker build --no-cache -t <username>/sharded-counter:latest . && docker push <username>/sharded-counter:latest
   ```

4. **Deploy to Kubernetes:**

   - Ensure your Kubernetes context is set correctly.
   - Apply the Kubernetes manifests:
     ```bash
       kubectl apply -f kubernetes/deployment_shard.yaml && kubectl apply -f kubernetes/deployment_app.yaml
     ```

## Usage

- **Increment a Counter:**

  ```bash
  curl -X PUT http://<app-server-ip>/counter/increment -d '{"counter_id": "example-counter"}'
  ```

- **Decrement a Counter:**

  ```bash
  curl -X PUT http://<app-server-ip>/counter/decrement -d '{"counter_id": "example-counter"}'
  ```

- **Get Counter Value:**

  ```bash
  curl http://<app-server-ip>/counter/value?counter_id=example-counter
  ```

## Benchmarking

### Tool Used

We used [Apache Benchmark (ab)](https://httpd.apache.org/docs/2.4/programs/ab.html) to evaluate the performance of the Sharded Counters application. Apache Benchmark helps simulate concurrent HTTP requests to test throughput, latency, and scalability.

### Test Environment

The benchmarking tests were performed on the following local machine configuration:
- **Operating System:** macOS 15.2 (24C101)
- **Processor:** Apple M2 8 Cores (4 performance and 4 efficiency)
- **RAM:** 8 GB
- **Disk:** 228 GiB
- **Network:** Localhost environment (no external network latency)

### Output Results

We used [Apache Benchmark (ab)](https://httpd.apache.org/docs/2.4/programs/ab.html) to evaluate the performance of the Sharded Counters application. Apache Benchmark helps simulate concurrent HTTP requests to test throughput, latency, and scalability.

### Output Results

#### Increment API

```bash
ab -n 1000 -c 100 -T "application/json" -p increment-payload.json -m PUT http://localhost:5000/counter/increment
```

**Sample Output:**

```
Concurrency Level:      100
Time taken for tests:   6.381 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      170000 bytes
Total body sent:        191000
HTML transferred:       62000 bytes
Requests per second:    156.71 [#/sec] (mean)
Time per request:       638.117 [ms] (mean)
Time per request:       6.381 [ms] (mean, across all concurrent requests)
Transfer rate:          26.02 [Kbytes/sec] received
                        29.23 kb/s sent
                        55.25 kb/s total
```


#### Decrement API

```bash
ab -n 1000 -c 100 -T "application/json" -p decrement-payload.json -m PUT http://localhost:5000/counter/decrement
```

**Sample Output:**

```
Concurrency Level:      100
Time taken for tests:   6.729 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      170000 bytes
Total body sent:        191000
HTML transferred:       62000 bytes
Requests per second:    148.60 [#/sec] (mean)
Time per request:       672.925 [ms] (mean)
Time per request:       6.729 [ms] (mean, across all concurrent requests)
Transfer rate:          24.67 [Kbytes/sec] received
                        27.72 kb/s sent
                        52.39 kb/s total
```

#### Get API

```bash
ab -n 1000 -c 50 ab -n 1000 -c 100 'http://localhost:5000/counter?counter_id=9999zzzzabchhhii'
```

**Sample Output:**

```
Concurrency Level:      100
Time taken for tests:   6.807 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      221000 bytes
HTML transferred:       112000 bytes
Requests per second:    146.91 [#/sec] (mean)
Time per request:       680.702 [ms] (mean)
Time per request:       6.807 [ms] (mean, across all concurrent requests)
Transfer rate:          31.71 [Kbytes/sec] received
```

### Notes

- **Requests per second**: Higher is better. It indicates how many requests the server can handle per second.
- **Time per request**: Lower is better. It reflects how quickly each request is processed.
- **Transfer rate**: Shows the data throughput during the test.

These benchmarks were performed on a local environment and may vary in production settings.

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Future Enhancements

1. **gRPC-based Communication:** Replace traditional HTTP-based communication between the app and shards with gRPC for enhanced performance, lower latency, and efficient serialization.
2. **Enhanced Fault Tolerance with Replication:** Integrate robust replication mechanisms for each shard to ensure high availability and seamless recovery from failures.
3. **Leader-Follower Architecture for Shards:** Adopt a leader-follower model primarily for fault tolerance, where the leader manages write operations, and followers act as hot standbys, ready to take over in case of leader failure.
4. **Scheduled Data Synchronization Across Shards:** Implement periodic data replication schedules to maintain consistent data integrity and availability across shards.
5. **Advanced Request Logging for Recovery:** Maintain detailed logs of all operations on the leader, enabling the replay of operations in case a follower needs to be promoted to a leader.
6. **Efficient Aggregated Count Storage:** Regularly compute and persist aggregated counts across all shards in a scalable storage solution, like Cassandra, to optimize on get counter value queries.