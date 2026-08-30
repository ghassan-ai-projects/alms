package server

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/service"
)

func registerProtocolsResource(mcpServer *server.MCPServer, learning *service.Learning) {
	mcpServer.AddResource(mcp.NewResource(
		"alms://protocols",
		"All Protocols",
		mcp.WithResourceDescription("List of all protocol records"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		protocols, err := learning.ProtocolList(ctx)
		if err != nil {
			return nil, fmt.Errorf("list protocols: %w", err)
		}

		return newJSONResourceContents("alms://protocols", protocols, "marshal protocols")
	})
}
