package server

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/service"
)

func registerLearningsResource(mcpServer *server.MCPServer, learning *service.Learning) {
	mcpServer.AddResource(mcp.NewResource(
		"alms://learnings",
		"All Learnings",
		mcp.WithResourceDescription("List of all active learning records"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		records, err := learning.Search(ctx, "", "", nil, 100)
		if err != nil {
			return nil, fmt.Errorf("list learnings: %w", err)
		}

		return newJSONResourceContents("alms://learnings", records, "marshal learnings")
	})
}
