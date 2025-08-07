package client

import (
	"context"

	promptsinternal "github.com/mycelian/mycelian-memory/clients/go/prompts"
)

// Prompt operations - all methods operate directly on Client

// LoadDefaultPrompts returns the default prompts for the given memory type ("chat", "code", ...).
func (c *Client) LoadDefaultPrompts(ctx context.Context, memoryType string) (*promptsinternal.DefaultPromptResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return promptsinternal.LoadDefaultPrompts(memoryType)
}
