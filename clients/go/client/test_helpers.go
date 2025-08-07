//go:build test
// +build test

package client

// setExecutor allows tests to replace the internal executor.
func (c *Client) setExecutor(exec executor) {
	c.exec = exec
}
