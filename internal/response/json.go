package response

import (
	"encoding/json"
	"log"
	"net/http"
)

// JSON writes a JSON response with the given status code and data.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("Failed to encode JSON response: %v", err)
		}
	}
}

// ErrorPayload is the standard shape for error responses.
type ErrorPayload struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// Error writes a JSON error response with the given status code and message.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, ErrorPayload{
		Error:   http.StatusText(status),
		Message: message,
	})
}
