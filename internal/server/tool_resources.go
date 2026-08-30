package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerToolsResource(mcpServer *server.MCPServer) {
	mcpServer.AddResource(mcp.NewResource(
		"alms://tools",
		"Tool Catalog",
		mcp.WithResourceDescription("All registered ALMS MCP tools"),
		mcp.WithMIMEType("application/json"),
	), readToolsResource)
}

func readToolsResource(context.Context, mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return newJSONResourceContents("alms://tools", toolCatalog(), "marshal tools")
}

func toolCatalog() []map[string]any {
	return []map[string]any{
		{"name": "agent.register", "description": "Register a new agent"},
		{"name": "agent.unregister", "description": "Unregister an existing agent"},
		{"name": "agent.update", "description": "Update an existing agent"},
		{"name": "agent.list", "description": "List registered agents"},
		{"name": "agent.heartbeat", "description": "Send a heartbeat for an agent"},
		{"name": "learning.sync", "description": "Sync new learnings for an agent"},
		{"name": "learning.sync_ack", "description": "Acknowledge received learnings (gap-safe)"},
		{"name": "learning.store", "description": "Create a new learning record with dedup check"},
		{"name": "learning.search", "description": "Full-text search for learnings via GIN index"},
		{"name": "learning.delete", "description": "Soft-delete a learning record"},
		{"name": "learning.get", "description": "Get a single learning record by ID"},
		{"name": "protocol.pull", "description": "Pull active protocols for an agent"},
		{"name": "protocol.pull_since", "description": "Pull protocols created after a given protocol ID"},
		{"name": "protocol.push", "description": "Create a new protocol"},
		{"name": "protocol.list", "description": "List all protocols"},
		{"name": "health.check", "description": "Check server health: PG ping + agent count"},
	}
}
