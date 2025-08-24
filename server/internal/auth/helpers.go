package auth

import (
	"errors"
	"net/http"
	"strings"
)

// ExtractAPIKey extracts API key from Authorization header
// Returns the API key or error if missing/invalid format
func ExtractAPIKey(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing Authorization header")
	}

	// Expect "Bearer <api_key>" format
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid Authorization header format, expected 'Bearer <api_key>'")
	}

	return parts[1], nil
}
