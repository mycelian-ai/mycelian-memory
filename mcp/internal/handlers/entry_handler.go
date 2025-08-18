package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	clientpkg "github.com/mycelian/mycelian-memory/client"
	"github.com/rs/zerolog/log"
)

// EntryHandler exposes add_entry, list_entries, and get_entry tools.
type EntryHandler struct {
	client *clientpkg.Client
}

// NewEntryHandler returns a new handler.
func NewEntryHandler(c *clientpkg.Client) *EntryHandler {
	return &EntryHandler{client: c}
}

const maxToolLimit = 50

// RegisterTools registers entry tools.
func (eh *EntryHandler) RegisterTools(s *server.MCPServer) error {
	// add_entry (vault scoped)
	addEntry := mcp.NewTool("add_entry",
		mcp.WithDescription("Append a new message or note to a memory inside a vault. RawEntry should contain the full text; Summary is a short recap."),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("The UUID of the user")),
		mcp.WithString("vault_id", mcp.Required(), mcp.Description("The UUID of the vault")),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("The UUID of the memory")),
		mcp.WithString("raw_entry", mcp.Required(), mcp.Description("Raw entry text")),
		mcp.WithString("summary", mcp.Required(), mcp.Description("Short summary of entry")),
		mcp.WithObject("tags", mcp.Description("Optional JSON object of tags")),
	)
	s.AddTool(addEntry, eh.handleAddEntry)

	// list_entries (vault scoped)
	listEntries := mcp.NewTool("list_entries",
		mcp.WithDescription("List entries for a memory within a vault with pagination cursors"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("The UUID of the user")),
		mcp.WithString("vault_id", mcp.Required(), mcp.Description("The UUID of the vault")),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("The UUID of the memory")),
		mcp.WithString("limit", mcp.Description("Max rows (1-50), default 25")),
		mcp.WithString("before", mcp.Description("Return entries created before this RFC3339 timestamp")),
		mcp.WithString("after", mcp.Description("Return entries created after this RFC3339 timestamp")),
	)
	s.AddTool(listEntries, eh.handleListEntries)

	// get_entry (vault scoped)
	getEntry := mcp.NewTool("get_entry",
		mcp.WithDescription("Get a single entry by entryId within a memory"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("The UUID of the user")),
		mcp.WithString("vault_id", mcp.Required(), mcp.Description("The UUID of the vault")),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("The UUID of the memory")),
		mcp.WithString("entry_id", mcp.Required(), mcp.Description("The UUID of the entry")),
	)
	s.AddTool(getEntry, eh.handleGetEntry)

	return nil
}

func (eh *EntryHandler) handleAddEntry(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, _ := req.RequireString("user_id")
	vaultID, _ := req.RequireString("vault_id")
	memoryID, _ := req.RequireString("memory_id")
	rawEntry, _ := req.RequireString("raw_entry")
	summary, _ := req.RequireString("summary")
	var tags map[string]string
	if t, ok := req.GetArguments()["tags"]; ok {
		_ = mapstructureDecode(t, &tags)
	}

	log.Debug().
		Str("user_id", userID).
		Str("vault_id", vaultID).
		Str("memory_id", memoryID).
		Int("raw_entry_len", len(rawEntry)).
		Str("summary", summary).
		Msg("handling add_entry request")

	start := time.Now()
	ack, err := eh.client.AddEntry(ctx, vaultID, memoryID, clientpkg.AddEntryRequest{
		RawEntry: rawEntry,
		Summary:  summary,
		Tags:     tags,
	})
	elapsed := time.Since(start)

	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("vault_id", vaultID).
			Str("memory_id", memoryID).
			Dur("elapsed", elapsed).
			Msg("add_entry failed")
		return mcp.NewToolResultError(fmt.Sprintf("failed to add entry: %v", err)), nil
	}

	log.Debug().
		Str("user_id", userID).
		Str("vault_id", vaultID).
		Str("memory_id", memoryID).
		Dur("elapsed", elapsed).
		Str("status", ack.Status).
		Msg("add_entry completed")

	return mcp.NewToolResultText("enqueued"), nil
}

func (eh *EntryHandler) handleListEntries(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, _ := req.RequireString("user_id")
	vaultID, _ := req.RequireString("vault_id")
	memoryID, _ := req.RequireString("memory_id")

	limitInt := 25
	if l, ok := req.GetArguments()["limit"].(float64); ok { // JSON numbers decoded as float64
		limitInt = int(l)
	}
	if limitInt <= 0 {
		limitInt = 25
	}
	if limitInt > maxToolLimit {
		limitInt = maxToolLimit
	}

	var before, after string
	if v, ok := req.GetArguments()["before"].(string); ok {
		before = v
	}
	if v, ok := req.GetArguments()["after"].(string); ok {
		after = v
	}
	if before != "" && after != "" {
		return mcp.NewToolResultError("provide only one of before or after"), nil
	}

	params := map[string]string{"limit": strconv.Itoa(limitInt)}
	if before != "" {
		params["before"] = before
	}
	if after != "" {
		params["after"] = after
	}

	log.Debug().
		Str("user_id", userID).
		Str("vault_id", vaultID).
		Str("memory_id", memoryID).
		Int("limit", limitInt).
		Str("before", before).
		Str("after", after).
		Msg("handling list_entries request")

	start := time.Now()
	resp, err := eh.client.ListEntries(ctx, vaultID, memoryID, params)
	elapsed := time.Since(start)

	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("vault_id", vaultID).
			Str("memory_id", memoryID).
			Dur("elapsed", elapsed).
			Msg("list_entries failed")
		return mcp.NewToolResultError(fmt.Sprintf("failed to list entries: %v", err)), nil
	}

	log.Debug().
		Str("user_id", userID).
		Str("vault_id", vaultID).
		Str("memory_id", memoryID).
		Dur("elapsed", elapsed).
		Int("count", resp.Count).
		Int("entries_returned", len(resp.Entries)).
		Msg("list_entries completed")

	// compute cursors
	var nextBefore, nextAfter *string
	if len(resp.Entries) == limitInt {
		oldest := resp.Entries[len(resp.Entries)-1].CreationTime
		ts := oldest.Format(time.RFC3339Nano)
		if after != "" {
			nextAfter = &ts
		} else {
			nextBefore = &ts
		}
	}
	payload := map[string]interface{}{
		"entries":       resp.Entries,
		"count":         resp.Count,
		"applied_limit": limitInt,
		"next_before":   nextBefore,
		"next_after":    nextAfter,
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}

func (eh *EntryHandler) handleGetEntry(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, _ := req.RequireString("user_id")
	vaultID, _ := req.RequireString("vault_id")
	memoryID, _ := req.RequireString("memory_id")
	entryID, _ := req.RequireString("entry_id")

	log.Debug().
		Str("user_id", userID).
		Str("vault_id", vaultID).
		Str("memory_id", memoryID).
		Str("entry_id", entryID).
		Msg("handling get_entry request")

	start := time.Now()
	e, err := eh.client.GetEntry(ctx, vaultID, memoryID, entryID)
	elapsed := time.Since(start)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", userID).
			Str("vault_id", vaultID).
			Str("memory_id", memoryID).
			Str("entry_id", entryID).
			Dur("elapsed", elapsed).
			Msg("get_entry failed")
		return mcp.NewToolResultError(fmt.Sprintf("failed to get entry: %v", err)), nil
	}

	b, _ := json.MarshalIndent(e, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}

// helper to decode generic map into typed struct
func mapstructureDecode(input interface{}, out interface{}) error {
	b, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}
