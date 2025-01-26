package counter_test

import (
	"math/rand"
	counter "sharded-counters/internal/shard_store"
	"testing"
	"time"
)

func TestConcurrentIncrementAndDecrement(t *testing.T) {
	manager := counter.GetCounterManager()
	counterID := "test-concurrent"

	const numGoroutines = 100
	const maxOperations = 1000

	// Channels to signal completion of increments and decrements
	incrementDone := make(chan int64, numGoroutines) // Send total operations per goroutine
	decrementDone := make(chan int64, numGoroutines) // Send total operations per goroutine

	// Create a new random generator with a seed
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Calculate dynamic timeout based on workload
	timeout := time.Duration(numGoroutines*maxOperations/1000) * time.Millisecond

	// Launch goroutines for increments
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			// Randomize the number of operations per goroutine
			operations := rng.Intn(maxOperations) + 1
			for j := 0; j < operations; j++ {
				manager.Increment(counterID)
			}
			t.Logf("Increment Goroutine %d completed %d operations", goroutineID, operations)
			incrementDone <- int64(operations) // Send total operations
		}(i)
	}

	// Launch goroutines for decrements
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			// Randomize the number of operations per goroutine
			operations := rng.Intn(maxOperations) + 1
			for j := 0; j < operations; j++ {
				manager.Decrement(counterID)
			}
			t.Logf("Decrement Goroutine %d completed %d operations", goroutineID, operations)
			decrementDone <- int64(operations) // Send total operations
		}(i)
	}

	// Wait for all increment goroutines and sum operations
	var totalIncrements int64
	for i := 0; i < numGoroutines; i++ {
		select {
		case ops := <-incrementDone:
			totalIncrements += ops
		case <-time.After(timeout):
			t.Fatalf("Timeout waiting for increment goroutines to finish")
		}
	}

	// Wait for all decrement goroutines and sum operations
	var totalDecrements int64
	for i := 0; i < numGoroutines; i++ {
		select {
		case ops := <-decrementDone:
			totalDecrements += ops
		case <-time.After(timeout):
			t.Fatalf("Timeout waiting for decrement goroutines to finish")
		}
	}

	// Verify that the final value matches the difference
	expectedValue := totalIncrements - totalDecrements
	actualValue := manager.Get(counterID)
	if actualValue != expectedValue {
		t.Errorf("Counter value mismatch: expected %d, got %d", expectedValue, actualValue)
	}
}
