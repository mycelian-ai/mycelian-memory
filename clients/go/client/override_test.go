package client

// overrideExecutor is test-only helper that swaps the client's executor.
func (c *Client) overrideExecutor(exec executor) { c.exec = exec }
