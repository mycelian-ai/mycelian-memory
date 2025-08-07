package api

import (
	"net/http"
)

// HTTPClient interface for dependency injection
// We'll import the client package as needed in specific API files
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
