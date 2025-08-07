package client

import (
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/rs/zerolog/log"
)

// debugTransport – optional HTTP round-trip logger
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

func debugLoggingRequested() bool {
	return os.Getenv("MYCELIAN_DEBUG") == "true" || os.Getenv("DEBUG") == "true"
}
