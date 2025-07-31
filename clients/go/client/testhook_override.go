//go:build test
// +build test

package client

// overrideExecutor allows tests to replace the internal executor.
func (c *Client) overrideExecutor(exec executor) {
	c.exec = exec
}
