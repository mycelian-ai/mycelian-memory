package client

// This file defines functional options that configure the Client during
// construction. Keeping them in a standalone file avoids cluttering
// client.go and makes it easy to discover all available knobs at a glance.

import (
	"fmt"
	"time"
)

// Option mutates the Client during New().
type Option func(*Client) error

// WithHTTPTimeout sets the HTTP client timeout. Keep initial surface minimal.
func WithHTTPTimeout(d time.Duration) Option {
	return func(c *Client) error {
		if d <= 0 {
			return fmt.Errorf("http timeout must be > 0")
		}
		c.http.Timeout = d
		return nil
	}
}

// WithDebugLogging wraps the client's transport such that every request/response
// is logged when `enabled` is true.
func WithDebugLogging(enabled bool) Option {
	return func(c *Client) error {
		if enabled {
			c.http.Transport = &debugTransport{base: c.http.Transport}
		}
		return nil
	}
}
