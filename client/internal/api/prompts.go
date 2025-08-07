package api

import (
	"context"

	promptsinternal "github.com/mycelian/mycelian-memory/client/prompts"
)

// DefaultPromptResponse is the JSON-serialisable structure returned to callers (defined locally)
type DefaultPromptResponse struct {
	Version             string            `json:"version"`
	ContextSummaryRules string            `json:"context_summary_rules"`
	Templates           map[string]string `json:"templates"`
}

// LoadDefaultPrompts returns the default prompts for the given memory type ("chat", "code", ...).
func LoadDefaultPrompts(ctx context.Context, memoryType string) (*DefaultPromptResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Delegate to the internal prompts package
	response, err := promptsinternal.LoadDefaultPrompts(memoryType)
	if err != nil {
		return nil, err
	}

	// Convert internal response to API response (they happen to be identical, but keep pattern)
	return &DefaultPromptResponse{
		Version:             response.Version,
		ContextSummaryRules: response.ContextSummaryRules,
		Templates:           response.Templates,
	}, nil
}
