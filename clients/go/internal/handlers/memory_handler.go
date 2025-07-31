package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
	"github.com/synapse/synapse-mcp-server/client"
)

// MemoryHandler provides memory management tools for the Memory service.
type MemoryHandler struct {
	client *client.Client
}

// NewMemoryHandler creates a new memory handler instance.
func NewMemoryHandler(client *client.Client) *MemoryHandler {
	return &MemoryHandler{
		client: client,
	}
}

// RegisterTools registers all memory management tools with the MCP server.
func (mh *MemoryHandler) RegisterTools(s *server.MCPServer) error {
	// get_memory – read-only
	getMem := mcp.NewTool("get_memory",
		mcp.WithDescription("Get memory details by ID inside a vault"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User UUID")),
		mcp.WithString("vault_id", mcp.Required(), mcp.Description("Vault UUID")),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("Memory UUID")),
	)

	s.AddTool(getMem, mh.handleGetMemory)

	// No create/list/update/delete tools exposed at user-level; use vault-scoped tool below.

	// NEW: create_memory_in_vault – write path that requires vault ID
	createMemInVault := mcp.NewTool("create_memory_in_vault",
		mcp.WithDescription("CAUTION: Use ONLY after the human has explicitly confirmed they want to create a new memory **inside a vault**. First ask for permission, then call this tool to create the memory for the given user and vault."),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User UUID")),
		mcp.WithString("vault_id", mcp.Required(), mcp.Description("Vault UUID")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Memory title (≤50 chars, lowercase/hyphen)")),
		mcp.WithString("memory_type", mcp.Required(), mcp.Description("Memory type e.g. NOTES, PROJECT")),
		mcp.WithString("description", mcp.Description("Optional memory description")),
	)

	s.AddTool(createMemInVault, mh.handleCreateMemoryInVault)

	return nil
}

func (mh *MemoryHandler) handleGetMemory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, _ := req.RequireString("user_id")
	vaultID, _ := req.RequireString("vault_id")
	memoryID, _ := req.RequireString("memory_id")

	log.Debug().
		Str("user_id", userID).
		Str("vault_id", vaultID).
		Str("memory_id", memoryID).
		Msg("handling get_memory request")

	start := time.Now()
	mem, err := mh.client.GetMemoryInVault(ctx, userID, vaultID, memoryID)
	elapsed := time.Since(start)

	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("vault_id", vaultID).
			Str("memory_id", memoryID).
			Dur("elapsed", elapsed).
			Msg("get_memory failed")
		return mcp.NewToolResultError(fmt.Sprintf("failed to get memory: %v", err)), nil
	}

	log.Debug().
		Str("user_id", userID).
		Str("vault_id", vaultID).
		Str("memory_id", memoryID).
		Str("title", mem.Title).
		Str("memory_type", mem.MemoryType).
		Dur("elapsed", elapsed).
		Msg("get_memory completed")

	b, _ := json.MarshalIndent(mem, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}

func (mh *MemoryHandler) handleCreateMemoryInVault(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, _ := req.RequireString("user_id")
	vaultID, _ := req.RequireString("vault_id")
	title, _ := req.RequireString("title")
	memoryType, _ := req.RequireString("memory_type")
	var description string
	if v, ok := req.GetArguments()["description"].(string); ok {
		description = v
	}

	log.Debug().
		Str("user_id", userID).
		Str("vault_id", vaultID).
		Str("title", title).
		Str("memory_type", memoryType).
		Str("description", description).
		Msg("handling create_memory_in_vault request")

	start := time.Now()
	mem, err := mh.client.CreateMemoryInVault(ctx, userID, vaultID, client.CreateMemoryRequest{
		Title:       title,
		MemoryType:  memoryType,
		Description: description,
	})
	elapsed := time.Since(start)

	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("vault_id", vaultID).
			Str("title", title).
			Dur("elapsed", elapsed).
			Msg("create_memory_in_vault failed")
		return mcp.NewToolResultError(fmt.Sprintf("failed to create memory in vault: %v", err)), nil
	}

	log.Debug().
		Str("user_id", userID).
		Str("vault_id", vaultID).
		Str("memory_id", mem.ID).
		Str("title", mem.Title).
		Str("memory_type", mem.MemoryType).
		Dur("elapsed", elapsed).
		Msg("create_memory_in_vault completed")

	b, _ := json.MarshalIndent(mem, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}
