package client

// This file defines functional options that configure the Client during
// construction. Keeping them in a standalone file avoids cluttering
// client.go and makes it easy to discover all available knobs at a glance.

import (
	"fmt"
	"net/http"
)

// Option mutates the Client during New().
type Option func(*Client) error

// WithHTTPClient injects a custom *http.Client. Useful for setting transport
// timeouts, tracing, custom TLS settings, etc.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		if hc == nil {
			return fmt.Errorf("nil http client")
		}
		c.http = hc
		return nil
	}
}

// WithDebugLogging wraps the client's transport such that every request/response
// is logged when `enabled` is true.
func WithDebugLogging(enabled bool) Option {
	return func(c *Client) error {
		if enabled {
			transport := c.http.Transport
			if transport == nil {
				transport = http.DefaultTransport
			}
			c.http.Transport = &debugTransport{base: transport}
		}
		return nil
	}
}

// WithoutExecutor disables the internal shardqueue executor – useful for
// short-lived CLIs that only call synchronous endpoints.
func WithoutExecutor() Option {
	return func(c *Client) error {
		c.exec = &noOpExecutor{}
		return nil
	}
}
