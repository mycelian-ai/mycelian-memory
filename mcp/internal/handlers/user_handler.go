package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mycelian/mycelian-memory/client"
	"github.com/rs/zerolog/log"
)

// UserHandler provides user management tools for the Memory service.
type UserHandler struct {
	client *client.Client
}

// NewUserHandler creates a new user handler instance.
func NewUserHandler(client *client.Client) *UserHandler {
	return &UserHandler{
		client: client,
	}
}

// RegisterTools registers all user management tools with the MCP server.
func (uh *UserHandler) RegisterTools(s *server.MCPServer) error {
	// Create get_user tool
	getUserTool := mcp.NewTool("get_user",
		mcp.WithDescription("Get detailed information about a specific user"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("The UUID of the user")),
	)
	s.AddTool(getUserTool, uh.handleGetUser)

	// user creation/update tools intentionally omitted — these operations are reserved for human‐facing CLI only.

	return nil
}

// handleGetUser handles the get_user tool call.
func (uh *UserHandler) handleGetUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, err := request.RequireString("user_id")
	if err != nil {
		log.Error().Err(err).Msg("user_id parameter validation failed")
		return mcp.NewToolResultError("user_id parameter is required"), nil
	}

	log.Debug().
		Str("user_id", userID).
		Msg("handling get_user request")

	start := time.Now()
	user, err := uh.client.GetUser(ctx, userID)
	elapsed := time.Since(start)

	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Dur("elapsed", elapsed).
			Msg("get_user failed")
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get user %s: %v", userID, err)), nil
	}

	log.Debug().
		Str("user_id", userID).
		Str("email", user.Email).
		Str("display_name", user.DisplayName).
		Str("time_zone", user.TimeZone).
		Dur("elapsed", elapsed).
		Msg("get_user completed")

	result := fmt.Sprintf("User Details:\n- ID: %s\n- Email: %s\n- Display Name: %s\n- Time Zone: %s\n- Created: %s",
		user.ID, user.Email, user.DisplayName, user.TimeZone, user.CreatedAt.Format("2006-01-02 15:04:05"))
	return mcp.NewToolResultText(result), nil
}
