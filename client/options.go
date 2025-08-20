package client

// This file defines functional options that configure the Client during
// construction. Keeping them in a standalone file avoids cluttering
// client.go and makes it easy to discover all available knobs at a glance.

import (
	"fmt"
	"time"
)

// Option configures a Client during construction in New.
//
// Options are applied before the authorization transport wrapper is installed,
// so transport-related options (like debug logging) will be placed underneath
// the API-key wrapper. Options must be deterministic and side-effect free.
type Option func(*Client) error

// WithHTTPTimeout sets the underlying http.Client Timeout used by the SDK.
//
// Prefer per-request context deadlines where possible; this timeout is a
// coarse safety net that bounds the total time spent on a single HTTP request
// (including connection, TLS handshake, redirects, and reading the response).
// The value must be greater than zero.
func WithHTTPTimeout(d time.Duration) Option {
	return func(c *Client) error {
		if d <= 0 {
			return fmt.Errorf("http timeout must be > 0")
		}
		c.http.Timeout = d
		return nil
	}
}

// WithDebugLogging wraps the client's transport so each request/response is
// logged when enabled is true.
//
// The debug transport is installed beneath the API-key wrapper; logs are
// emitted before the request is forwarded to the next transport.
// Do not enable this option in production environments as it increases
// verbosity and may include headers and method/URL metadata in logs.
func WithDebugLogging(enabled bool) Option {
	return func(c *Client) error {
		if enabled {
			c.http.Transport = &debugTransport{base: c.http.Transport}
		}
		return nil
	}
}
