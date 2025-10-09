package respond

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mycelian/mycelian-memory/server/internal/storage"
	"github.com/rs/zerolog/log"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

// WriteJSON writes a JSON response with the given status code
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().Err(err).Msg("Failed to encode JSON response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// WriteError writes a standardized error response
func WriteError(w http.ResponseWriter, statusCode int, message string) {
	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Code:    statusCode,
		Message: message,
	}
	WriteJSON(w, statusCode, response)
}

// WriteBadRequest writes a 400 Bad Request response
func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, message)
}

// WriteNotFound writes a 404 Not Found response
func WriteNotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, message)
}

// WriteInternalError writes a 500 Internal Server Error response
func WriteInternalError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusInternalServerError, message)
}

// HandleError writes an appropriate HTTP response based on the error type.
// It inspects the error and maps storage layer errors to appropriate HTTP status codes:
//   - storage.ErrNotFound → 404 Not Found (uses defaultMessage)
//   - storage.ErrConflict → 409 Conflict (uses defaultMessage)
//   - storage.ErrNotImplemented → 501 Not Implemented (uses defaultMessage)
//   - all other errors → 500 Internal Server Error (generic message to avoid leaking internals)
func HandleError(w http.ResponseWriter, err error, defaultMessage string) {
	if err == nil {
		WriteInternalError(w, "unexpected nil error")
		return
	}

	switch {
	case errors.Is(err, storage.ErrNotFound):
		WriteNotFound(w, defaultMessage)
	case errors.Is(err, storage.ErrConflict):
		WriteError(w, http.StatusConflict, defaultMessage)
	case errors.Is(err, storage.ErrNotImplemented):
		WriteError(w, http.StatusNotImplemented, defaultMessage)
	default:
		// For genuine internal errors, use a generic message to avoid
		// exposing internal details or misleading with resource-specific text
		WriteInternalError(w, "internal server error")
	}
}
