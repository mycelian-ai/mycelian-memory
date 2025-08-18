//go:build integration
// +build integration

package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	mclient "github.com/mycelian/mycelian-memory/client"
	"github.com/mycelian/mycelian-memory/mcp/internal/handlers"
)

// TestMCPServerTransports verifies that the MCP server correctly serves tools
// over both in-process (stdio-like) and HTTP transports
func TestMCPServerTransports(t *testing.T) {
	// Create stub memory service backend
	memSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Minimal stub - we only need server startup, not actual calls
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer memSrv.Close()

	// Create MCP server with all handlers registered
	mcpServer := server.NewMCPServer(
		"test-mcp-server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Initialize client SDK pointing to stub backend
	sdk := mclient.New(memSrv.URL)

	// Register all handlers
	handlers := []struct {
		name    string
		handler interface{ RegisterTools(*server.MCPServer) error }
	}{
		{"memory", handlers.NewMemoryHandler(sdk)},
		{"entry", handlers.NewEntryHandler(sdk)},
		{"search", handlers.NewSearchHandler(sdk)},
		{"vault", handlers.NewVaultHandler(sdk)},
		{"context", handlers.NewContextHandler(sdk)},
		{"consistency", handlers.NewConsistencyHandler(sdk)},
	}

	for _, h := range handlers {
		if err := h.handler.RegisterTools(mcpServer); err != nil {
			t.Fatalf("failed to register %s tools: %v", h.name, err)
		}
	}

	// Test 1: In-process transport (simulates stdio)
	t.Run("InProcessTransport", func(t *testing.T) {
		inProcessTransport := transport.NewInProcessTransport(mcpServer)
		if err := inProcessTransport.Start(context.Background()); err != nil {
			t.Fatalf("failed to start in-process transport: %v", err)
		}
		defer inProcessTransport.Close()

		mcpClient := client.NewClient(inProcessTransport)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Initialize the MCP client
		_, err := mcpClient.Initialize(ctx, mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: "2024-11-05",
				Capabilities:    mcp.ClientCapabilities{},
				ClientInfo: mcp.Implementation{
					Name:    "test-client",
					Version: "1.0.0",
				},
			},
		})
		if err != nil {
			t.Fatalf("failed to initialize MCP client: %v", err)
		}

		// Call tools/list
		tools, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			t.Fatalf("tools/list failed over in-process transport: %v", err)
		}

		if len(tools.Tools) == 0 {
			t.Fatal("expected at least one tool, got none")
		}

		// Verify we have some expected tools
		toolNames := make(map[string]bool)
		for _, tool := range tools.Tools {
			toolNames[tool.Name] = true
		}

		expectedTools := []string{"get_user", "add_entry", "search_memories", "put_context"}
		for _, expected := range expectedTools {
			if !toolNames[expected] {
				t.Errorf("expected tool %q not found in tools list", expected)
			}
		}

		t.Logf("in-process transport: found %d tools", len(tools.Tools))
	})

	// Test 2: HTTP transport (streamable)
	t.Run("HTTPTransport", func(t *testing.T) {
		// Create streamable HTTP server
		streamSrv := server.NewStreamableHTTPServer(
			mcpServer,
			server.WithEndpointPath("/mcp"),
			server.WithHeartbeatInterval(30*time.Second),
		)

		httpSrv := httptest.NewServer(streamSrv)
		defer httpSrv.Close()

		// Create HTTP client
		httpTransport, err := transport.NewStreamableHTTP(httpSrv.URL + "/mcp")
		if err != nil {
			t.Fatalf("failed to create HTTP transport: %v", err)
		}
		if err := httpTransport.Start(context.Background()); err != nil {
			t.Fatalf("failed to start HTTP transport: %v", err)
		}
		defer httpTransport.Close()

		mcpClient := client.NewClient(httpTransport)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Initialize the MCP client
		_, initErr := mcpClient.Initialize(ctx, mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: "2024-11-05",
				Capabilities:    mcp.ClientCapabilities{},
				ClientInfo: mcp.Implementation{
					Name:    "test-client",
					Version: "1.0.0",
				},
			},
		})
		if initErr != nil {
			t.Fatalf("failed to initialize MCP client: %v", initErr)
		}

		// Call tools/list
		tools, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			t.Fatalf("tools/list failed over HTTP transport: %v", err)
		}

		if len(tools.Tools) == 0 {
			t.Fatal("expected at least one tool, got none")
		}

		t.Logf("HTTP transport: found %d tools", len(tools.Tools))
	})

	t.Logf("MCP server transport test completed successfully")
}
