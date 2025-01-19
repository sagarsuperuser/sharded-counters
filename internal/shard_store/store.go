package counter

import (
	"sync"
)

// Counter represents a single counter with its own lock.
type Counter struct {
	Value int64
	Lock  sync.Mutex
}

// CounterManager manages in-memory counters with granular locking.
type CounterManager struct {
	counters sync.Map // Thread-safe storage for counters.
}

var (
	instance     *CounterManager
	instanceOnce sync.Once
)

// GetCounterManager returns the singleton instance of CounterManager.
func GetCounterManager() *CounterManager {
	instanceOnce.Do(func() {
		instance = &CounterManager{}
	})
	return instance
}

// Increment increments the counter for the given ID with a granular lock.
func (cm *CounterManager) Increment(counterID string) int64 {
	// Load or create the counter.
	counter, _ := cm.counters.LoadOrStore(counterID, &Counter{})

	// Cast to the Counter type.
	c := counter.(*Counter)

	// Lock the specific counter and increment.
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.Value++
	return c.Value
}

// Decrement decrements the counter for the given ID with a granular lock.
func (cm *CounterManager) Decrement(counterID string) int64 {
	// Load or create the counter.
	counter, _ := cm.counters.LoadOrStore(counterID, &Counter{})

	// Cast to the Counter type.
	c := counter.(*Counter)

	// Lock the specific counter and increment.
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.Value--
	return c.Value
}

// Get retrieves the current value of a counter.
func (cm *CounterManager) Get(counterID string) int64 {
	// Load the counter if it exists.
	if counter, ok := cm.counters.Load(counterID); ok {
		return counter.(*Counter).Value
	}
	return 0 // Default value if counter doesn't exist.
}
