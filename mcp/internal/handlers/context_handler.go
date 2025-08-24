package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mycelian/mycelian-memory/client"
	"github.com/rs/zerolog/log"
)

// ContextHandler exposes put_context and get_context tools.
type ContextHandler struct {
	client *client.Client
}

// NewContextHandler returns a new handler.
func NewContextHandler(c *client.Client) *ContextHandler {
	return &ContextHandler{client: c}
}

// RegisterTools registers context tools with the MCP server.
func (ch *ContextHandler) RegisterTools(s *server.MCPServer) error {
	// put_context (vault scoped)
	putCtx := mcp.NewTool("put_context",
		mcp.WithDescription("Persist the single, plain-text context document for a memory inside a vault"),
		mcp.WithString("vault_id", mcp.Required(), mcp.Description("Vault UUID")),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("Memory UUID")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Raw context text (entire document)")),
	)
	s.AddTool(putCtx, ch.handlePutContext)

	// get_context (vault scoped)
	getCtx := mcp.NewTool("get_context",
		mcp.WithDescription("Fetch the full plain-text context document for a memory inside a vault"),
		mcp.WithString("vault_id", mcp.Required(), mcp.Description("Vault UUID")),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("Memory UUID")),
	)
	s.AddTool(getCtx, ch.handleGetContext)

	return nil
}

func (ch *ContextHandler) handlePutContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	vaultID, _ := req.RequireString("vault_id")
	memID, _ := req.RequireString("memory_id")
	content, _ := req.RequireString("content")

	log.Debug().
		Str("vault_id", vaultID).
		Str("memory_id", memID).
		Int("content_len", len(content)).
		Msg("handling put_context request")

	start := time.Now()
	// Use background context for async job to prevent cancellation when the tool call completes.
	jobCtx := context.Background()

	ack, err := ch.client.PutContext(jobCtx, vaultID, memID, content)
	elapsed := time.Since(start)

	if err != nil {
		log.Error().
			Err(err).
			Str("vault_id", vaultID).
			Str("memory_id", memID).
			Dur("elapsed", elapsed).
			Msg("put_context failed")
		return mcp.NewToolResultError(fmt.Sprintf("failed to put context: %v", err)), nil
	}

	log.Debug().
		Str("vault_id", vaultID).
		Str("memory_id", memID).
		Str("status", ack.Status).
		Dur("elapsed", elapsed).
		Msg("put_context completed")

	return mcp.NewToolResultText("enqueued"), nil
}

func (ch *ContextHandler) handleGetContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	vaultID, _ := req.RequireString("vault_id")
	memID, _ := req.RequireString("memory_id")

	log.Debug().
		Str("vault_id", vaultID).
		Str("memory_id", memID).
		Msg("handling get_context request")

	start := time.Now()
	text, err := ch.client.GetLatestContext(ctx, vaultID, memID)
	elapsed := time.Since(start)

	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			log.Debug().
				Str("vault_id", vaultID).
				Str("memory_id", memID).
				Dur("elapsed", elapsed).
				Msg("get_context: context not found")
			return mcp.NewToolResultError("context not found"), nil
		}
		log.Error().
			Err(err).
			Str("vault_id", vaultID).
			Str("memory_id", memID).
			Dur("elapsed", elapsed).
			Msg("get_context failed")
		return mcp.NewToolResultError(fmt.Sprintf("failed to get context: %v", err)), nil
	}

	log.Debug().
		Str("vault_id", vaultID).
		Str("memory_id", memID).
		Int("response_len", len(text)).
		Msg("get_context completed")

	return mcp.NewToolResultText(text), nil
}
