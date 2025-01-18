package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sharded-counters/internal/etcd"
	counter "sharded-counters/internal/shard_store"
	"time"
)

type Dependencies struct {
	CounterManager *counter.CounterManager
	EtcdManager    *etcd.EtcdManager
	// Add other dependencies as needed.
}

type contextKey string

const dependenciesKey contextKey = "dependencies"

// Middleware wraps an HTTP handler to log requests and injects EtcdManager into context.
func Middleware(deps *Dependencies, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Request started: %s %s %s", r.Method, r.RequestURI, start.Format(time.RFC3339))

		// Add dependencies to the context.
		ctx := context.WithValue(r.Context(), dependenciesKey, deps)
		r = r.WithContext(ctx)

		// Create a response recorder to capture response details.
		recorder := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)

		latency := time.Since(start)
		log.Printf(
			"Request completed: %s %s %s | Status: %d | Latency: %s",
			r.Method, r.RequestURI, start.Format(time.RFC3339), recorder.statusCode, latency,
		)
	})
}

// GetDependenciesFromContext retrieves the dependencies from the request context.
func GetDependenciesFromContext(ctx context.Context) (*Dependencies, error) {
	deps, ok := ctx.Value(dependenciesKey).(*Dependencies)
	if !ok {
		return nil, fmt.Errorf("dependencies not found in context")
	}
	return deps, nil
}

// responseRecorder wraps an http.ResponseWriter to capture status codes.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}
