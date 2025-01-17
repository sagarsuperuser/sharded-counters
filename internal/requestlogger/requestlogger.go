package requestlogger

import (
	"log"
	"net/http"
	"time"
)

// Middleware wraps an HTTP handler to log requests and responses.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Request started: %s %s %s", r.Method, r.RequestURI, start.Format(time.RFC3339))

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

// responseRecorder wraps an http.ResponseWriter to capture status codes.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}
