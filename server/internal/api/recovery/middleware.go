package recovery

import (
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog/log"
)

// Middleware intercepts panics from downstream handlers, logs details, and returns HTTP 500.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error().
					Interface("panic", rec).
					Str("method", r.Method).
					Str("url", r.URL.String()).
					Str("remote", r.RemoteAddr).
					Bytes("stack", debug.Stack()).
					Msg("panic recovered")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"Internal Server Error","code":500}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
