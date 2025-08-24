package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mycelian/mycelian-memory/client"
	"github.com/rs/zerolog/log"
)

// VaultHandler exposes vault-level management tools.
type VaultHandler struct {
	client *client.Client
}

func NewVaultHandler(c *client.Client) *VaultHandler { return &VaultHandler{client: c} }

func (vh *VaultHandler) RegisterTools(s *server.MCPServer) error {
	// create_vault – must be called before creating memories
	create := mcp.NewTool("create_vault",
		mcp.WithDescription("Create a vault for organising memories; returns vaultId and title"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Vault title (≤50 chars, lowercase/hyphen)")),
		mcp.WithString("description", mcp.Description("Optional vault description")),
	)

	// list_vaults – returns id + title for all vaults
	listVaults := mcp.NewTool("list_vaults",
		mcp.WithDescription("List all vaults (returns vaultId & title)"),
	)
	list := mcp.NewTool("list_memories",
		mcp.WithDescription("List memories inside a vault (returns id & title)"),
		mcp.WithString("vault_id", mcp.Required(), mcp.Description("Vault UUID")),
	)
	s.AddTool(create, vh.handleCreateVault)
	s.AddTool(listVaults, vh.handleListVaults)
	s.AddTool(list, vh.handleListMemories)
	return nil
}

func (vh *VaultHandler) handleCreateVault(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, _ := req.RequireString("title")
	var desc string
	if v, ok := req.GetArguments()["description"].(string); ok {
		desc = v
	}

	log.Debug().Str("title", title).Msg("create_vault invoked")

	start := time.Now()
	v, err := vh.client.CreateVault(ctx, client.CreateVaultRequest{Title: title, Description: desc})
	elapsed := time.Since(start)
	if err != nil {
		log.Error().Err(err).Dur("elapsed", elapsed).Msg("create_vault failed")
		return mcp.NewToolResultError(fmt.Sprintf("failed to create vault: %v", err)), nil
	}

	out := map[string]any{"vaultId": v.VaultID, "title": v.Title}
	b, _ := json.Marshal(out)
	return mcp.NewToolResultText(string(b)), nil
}

// handleListVaults returns a minimal list of vault identifiers and titles.
func (vh *VaultHandler) handleListVaults(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debug().Msg("list_vaults invoked")

	start := time.Now()
	vaults, err := vh.client.ListVaults(ctx)
	elapsed := time.Since(start)
	if err != nil {
		log.Error().Err(err).Dur("elapsed", elapsed).Msg("list_vaults failed")
		return mcp.NewToolResultError(fmt.Sprintf("failed to list vaults: %v", err)), nil
	}

	type lite struct {
		VaultID string `json:"vaultId"`
		Title   string `json:"title"`
	}
	out := make([]lite, len(vaults))
	for i, v := range vaults {
		out[i] = lite{VaultID: v.VaultID, Title: v.Title}
	}
	b, _ := json.Marshal(out)
	return mcp.NewToolResultText(string(b)), nil
}

func (vh *VaultHandler) handleListMemories(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	vaultID, _ := req.RequireString("vault_id")

	log.Debug().Str("vault_id", vaultID).Msg("list_memories invoked")

	start := time.Now()
	mems, err := vh.client.ListMemories(ctx, vaultID)
	elapsed := time.Since(start)
	if err != nil {
		log.Error().Err(err).Dur("elapsed", elapsed).Msg("list_memories failed")
		return mcp.NewToolResultError(fmt.Sprintf("failed to list memories: %v", err)), nil
	}

	// reduce to id + title
	type lite struct {
		MemoryID string `json:"memoryId"`
		Title    string `json:"title"`
	}
	out := make([]lite, len(mems))
	for i, m := range mems {
		out[i] = lite{MemoryID: m.ID, Title: m.Title}
	}
	b, _ := json.Marshal(out)
	return mcp.NewToolResultText(string(b)), nil
}
