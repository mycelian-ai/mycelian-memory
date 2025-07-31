package handlers

import (
	"reflect"
	"sort"
	"testing"

	"github.com/mark3labs/mcp-go/server"
	"github.com/synapse/synapse-mcp-server/client"
)

func TestToolCatalogue(t *testing.T) {
	// Build minimal MCP server.
	s := server.NewMCPServer("test", "dev", server.WithToolCapabilities(true))

	stubClient := client.New("http://stub")

	// Register all handlers.
	_ = NewUserHandler(stubClient).RegisterTools(s)
	_ = NewMemoryHandler(stubClient).RegisterTools(s)
	_ = NewEntryHandler(stubClient).RegisterTools(s)
	_ = NewContextHandler(stubClient).RegisterTools(s)
	_ = NewVaultHandler(stubClient).RegisterTools(s)
	_ = NewConsistencyHandler(stubClient).RegisterTools(s)
	_ = NewSearchHandler(stubClient).RegisterTools(s)

	// Access private field 'tools' via reflection to collect names.
	v := reflect.ValueOf(s).Elem().FieldByName("tools")
	if !v.IsValid() {
		t.Fatalf("failed to access tools map via reflection; server internals changed")
	}
	iter := v.MapRange()
	var got []string
	for iter.Next() {
		got = append(got, iter.Key().String())
	}
	sort.Strings(got)

	want := []string{
		"add_entry",
		"await_consistency",
		"create_memory_in_vault",
		"create_vault",
		"get_context",
		"get_memory",
		"get_user",
		"list_entries",
		"list_memories",
		"list_vaults",
		"put_context",
		"search_memories",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tool catalogue mismatch\nwant: %v\n got: %v", want, got)
	}
}
