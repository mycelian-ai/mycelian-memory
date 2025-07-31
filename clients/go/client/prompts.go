package client

import (
	"context"

	"github.com/synapse/synapse-mcp-server/prompts"
)

// GetDefaultPrompts returns the embedded default prompt templates for the given
// memoryType (e.g. "chat", "code"). It does not perform any network I/O – all
// assets are compiled into the binary at build time.
func (c *Client) GetDefaultPrompts(ctx context.Context, memoryType string) (*prompts.DefaultPromptResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return prompts.LoadDefaultPrompts(memoryType)
}
