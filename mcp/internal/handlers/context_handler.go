package handlers

import (
	"context"
	"encoding/json"
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
		mcp.WithDescription("Persist the activeContext document for a memory inside a vault"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User UUID")),
		mcp.WithString("vault_id", mcp.Required(), mcp.Description("Vault UUID")),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("Memory UUID")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Raw context text")),
	)
	s.AddTool(putCtx, ch.handlePutContext)

	// get_context (vault scoped)
	getCtx := mcp.NewTool("get_context",
		mcp.WithDescription("Fetch specific fragments of the current context document for a memory inside a vault"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User UUID")),
		mcp.WithString("vault_id", mcp.Required(), mcp.Description("Vault UUID")),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("Memory UUID")),
		mcp.WithArray("fragments", mcp.Description("Optional list of top-level keys to return")),
	)
	s.AddTool(getCtx, ch.handleGetContext)

	return nil
}

func (ch *ContextHandler) handlePutContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, _ := req.RequireString("user_id")
	vaultID, _ := req.RequireString("vault_id")
	memID, _ := req.RequireString("memory_id")
	content, _ := req.RequireString("content")

	log.Debug().
		Str("user_id", userID).
		Str("vault_id", vaultID).
		Str("memory_id", memID).
		Int("content_len", len(content)).
		Msg("handling put_context request")

	start := time.Now()
	ack, err := ch.client.PutContext(ctx, vaultID, memID, client.PutContextRequest{Context: content})
	elapsed := time.Since(start)

	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("vault_id", vaultID).
			Str("memory_id", memID).
			Dur("elapsed", elapsed).
			Msg("put_context failed")
		return mcp.NewToolResultError(fmt.Sprintf("failed to put context: %v", err)), nil
	}

	log.Debug().
		Str("user_id", userID).
		Str("vault_id", vaultID).
		Str("memory_id", memID).
		Str("status", ack.Status).
		Dur("elapsed", elapsed).
		Msg("put_context completed")

	return mcp.NewToolResultText("enqueued"), nil
}

func (ch *ContextHandler) handleGetContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, _ := req.RequireString("user_id")
	vaultID, _ := req.RequireString("vault_id")
	memID, _ := req.RequireString("memory_id")

	// optional fragments argument
	var wanted []string
	if v, ok := req.GetArguments()["fragments"].([]interface{}); ok {
		for _, it := range v {
			if s, ok2 := it.(string); ok2 {
				wanted = append(wanted, s)
			}
		}
	}

	log.Debug().
		Str("user_id", userID).
		Str("vault_id", vaultID).
		Str("memory_id", memID).
		Strs("fragments", wanted).
		Msg("handling get_context request")

	start := time.Now()
	res, err := ch.client.GetContext(ctx, vaultID, memID)
	elapsed := time.Since(start)

	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			log.Debug().
				Str("user_id", userID).
				Str("vault_id", vaultID).
				Str("memory_id", memID).
				Dur("elapsed", elapsed).
				Msg("get_context: context not found")
			return mcp.NewToolResultError("context not found"), nil
		}
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("vault_id", vaultID).
			Str("memory_id", memID).
			Dur("elapsed", elapsed).
			Msg("get_context failed")
		return mcp.NewToolResultError(fmt.Sprintf("failed to get context: %v", err)), nil
	}

	obj, ok := res.Context.(map[string]interface{})
	if !ok {
		raw, _ := json.Marshal(res.Context)
		log.Debug().
			Str("user_id", userID).
			Str("vault_id", vaultID).
			Str("memory_id", memID).
			Dur("elapsed", elapsed).
			Int("response_len", len(raw)).
			Str("type", "raw").
			Msg("get_context completed")
		return mcp.NewToolResultText(string(raw)), nil
	}

	// if no fragment list specified, return whole object
	if len(wanted) == 0 {
		raw, _ := json.Marshal(obj)
		log.Debug().
			Str("user_id", userID).
			Str("vault_id", vaultID).
			Str("memory_id", memID).
			Dur("elapsed", elapsed).
			Int("response_len", len(raw)).
			Int("total_keys", len(obj)).
			Str("type", "full").
			Msg("get_context completed")
		return mcp.NewToolResultText(string(raw)), nil
	}

	filtered := make(map[string]interface{})
	for _, k := range wanted {
		if val, ok := obj[k]; ok {
			filtered[k] = val
		}
	}
	raw, _ := json.Marshal(filtered)

	log.Debug().
		Str("user_id", userID).
		Str("vault_id", vaultID).
		Str("memory_id", memID).
		Dur("elapsed", elapsed).
		Int("response_len", len(raw)).
		Int("total_keys", len(obj)).
		Int("filtered_keys", len(filtered)).
		Strs("requested_fragments", wanted).
		Str("type", "filtered").
		Msg("get_context completed")

	return mcp.NewToolResultText(string(raw)), nil
}
