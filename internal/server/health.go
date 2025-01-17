package server

import (
	"net/http"
	"sharded-counters/internal/responsehandler"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	// log.Printf("Hey I am Healthy")
	responsehandler.SendSuccessResponse(w, "Hey I am Healthy", nil)
}
