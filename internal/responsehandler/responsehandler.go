package responsehandler

import (
	"encoding/json"
	"net/http"
)

// Response represents the structure of an API response.
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError represents the structure of an API error.
type APIError struct {
	Code    int    `json:"code"`
	Details string `json:"details"`
}

// SendSuccessResponse sends a successful JSON response.
func SendSuccessResponse(w http.ResponseWriter, message string, data interface{}) {
	response := Response{
		Success: true,
		Message: message,
		Data:    data,
	}
	sendJSONResponse(w, http.StatusOK, response)
}

// SendErrorResponse sends an error JSON response.
func SendErrorResponse(w http.ResponseWriter, code int, message string, details string) {
	response := Response{
		Success: false,
		Message: message,
		Error: &APIError{
			Code:    code,
			Details: details,
		},
	}
	sendJSONResponse(w, code, response)
}

// sendJSONResponse sends a JSON response with the given status code and payload.
func sendJSONResponse(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
