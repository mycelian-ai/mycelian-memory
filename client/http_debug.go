package client

import (
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/rs/zerolog/log"
)

// debugTransport provides detailed HTTP request/response logging for debugging client issues.
//
// Purpose:
//   - Troubleshoot API communication problems (timeouts, malformed requests, unexpected responses)
//   - Debug authentication/authorization issues by inspecting headers and payloads
//   - Analyze performance bottlenecks by examining request/response sizes and timing
//   - Validate request formatting during development and testing
//
// When to use:
//   - Set MYCELIAN_DEBUG=true or DEBUG=true environment variable
//   - During development when building new API integrations
//   - When investigating production issues (temporarily, with log level controls)
//   - In CI/CD pipelines for integration test debugging
//
// Security considerations:
//   - Logs full request/response bodies including sensitive data (tokens, user data)
//   - Only enable in development/staging environments
//   - Ensure log outputs are properly secured and not exposed
//
// Performance impact:
//   - Adds overhead for request/response dumping and logging
//   - Should be disabled in production for optimal performance
//
// Example usage:
//
//	export MYCELIAN_DEBUG=true
//	go run main.go  # Client will now log all HTTP traffic
type debugTransport struct{ base http.RoundTripper }

func (dt *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if debugLoggingRequested() {
		if reqDump, err := httputil.DumpRequestOut(req, true); err == nil {
			log.Debug().Str("method", req.Method).Str("url", req.URL.String()).Str("request_dump", string(reqDump)).Msg("HTTP request")
		}
	}

	resp, err := dt.base.RoundTrip(req)
	if err != nil {
		if debugLoggingRequested() {
			log.Error().Err(err).Str("method", req.Method).Str("url", req.URL.String()).Msg("HTTP request failed")
		}
		return nil, err
	}

	if debugLoggingRequested() {
		if respDump, err := httputil.DumpResponse(resp, true); err == nil {
			log.Debug().Str("method", req.Method).Str("url", req.URL.String()).Int("status_code", resp.StatusCode).Str("response_dump", string(respDump)).Msg("HTTP response")
		}
	}
	return resp, nil
}

// debugLoggingRequested checks if HTTP debug logging should be enabled.
//
// Activation methods:
//   - MYCELIAN_DEBUG=true (mycelian-specific debug flag)
//   - DEBUG=true (general debug flag, common in development workflows)
//
// Both environment variables are supported for flexibility:
//   - Use MYCELIAN_DEBUG for targeted mycelian client debugging
//   - Use DEBUG for broader application debugging that includes HTTP traffic
//
// Returns true if either environment variable is set to "true" (case-sensitive).
func debugLoggingRequested() bool {
	return os.Getenv("MYCELIAN_DEBUG") == "true" || os.Getenv("DEBUG") == "true"
}
