package http

import (
	"github.com/gorilla/mux"
)

// NewRouter creates a router. Callers should register handlers.
func NewRouter() *mux.Router {
	r := mux.NewRouter()
	return r
}
