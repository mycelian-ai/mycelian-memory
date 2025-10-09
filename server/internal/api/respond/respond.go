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

// WriteInternalError writes an HTTP 500 Internal Server Error response with the provided message.
func WriteInternalError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusInternalServerError, message)
}

// HandleError writes an appropriate HTTP response based on the error type.
// It inspects the error and maps storage layer errors to appropriate HTTP status codes:
//   - storage.ErrNotFound → 404 Not Found
//   - storage.ErrConflict → 409 Conflict
//   - storage.ErrNotImplemented → 501 Not Implemented
// HandleError maps an error to an appropriate HTTP error response and writes it to w.
// If err is nil, it writes a 500 Internal Server Error with the message "unexpected nil error".
// For non-nil errors it writes:
//  - 404 Not Found when err matches storage.ErrNotFound,
//  - 409 Conflict when err matches storage.ErrConflict,
//  - 501 Not Implemented when err matches storage.ErrNotImplemented,
//  - 500 Internal Server Error for all other errors.
// The provided defaultMessage is used as the response message for mapped errors.
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
		WriteInternalError(w, defaultMessage)
	}
}