package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/ghassan/alms/internal/service"
)

// registerResources registers all MCP resource handlers on the server.
func (s *Server) registerResources(registry *service.Registry, learning *service.Learning) {
	// alms://agents — list all agents
	s.mcp.AddResource(mcp.NewResource(
		"alms://agents",
		"All Agents",
		mcp.WithResourceDescription("List of all registered agents"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		agents, err := registry.List(ctx, nil, 100, 0)
		if err != nil {
			return nil, fmt.Errorf("list agents: %w", err)
		}

		data, err := json.Marshal(agents)
		if err != nil {
			return nil, fmt.Errorf("marshal agents: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "alms://agents",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// alms://health — server health
	s.mcp.AddResource(mcp.NewResource(
		"alms://health",
		"Server Health",
		mcp.WithResourceDescription("ALMS server health status"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		agentCount, err := registry.AgentCount(ctx)
		if err != nil {
			agentCount = -1
		}

		health := map[string]any{
			"status":      "ok",
			"version":     "0.1.0",
			"agent_count": agentCount,
		}

		data, err := json.Marshal(health)
		if err != nil {
			return nil, fmt.Errorf("marshal health: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "alms://health",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// alms://learnings — list all learnings (Phase 2)
	s.mcp.AddResource(mcp.NewResource(
		"alms://learnings",
		"All Learnings",
		mcp.WithResourceDescription("List of all active learning records"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		records, err := learning.Search(ctx, "", "", nil, 100)
		if err != nil {
			return nil, fmt.Errorf("list learnings: %w", err)
		}

		data, err := json.Marshal(records)
		if err != nil {
			return nil, fmt.Errorf("marshal learnings: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "alms://learnings",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// alms://tools — aggregated tool catalog (Phase 2)
	s.mcp.AddResource(mcp.NewResource(
		"alms://tools",
		"Tool Catalog",
		mcp.WithResourceDescription("All registered ALMS MCP tools"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		tools := []map[string]any{
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

		data, err := json.Marshal(tools)
		if err != nil {
			return nil, fmt.Errorf("marshal tools: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "alms://tools",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// alms://protocols — list protocols (Phase 2)
	s.mcp.AddResource(mcp.NewResource(
		"alms://protocols",
		"All Protocols",
		mcp.WithResourceDescription("List of all protocol records"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		protocols, err := learning.ProtocolList(ctx)
		if err != nil {
			return nil, fmt.Errorf("list protocols: %w", err)
		}

		data, err := json.Marshal(protocols)
		if err != nil {
			return nil, fmt.Errorf("marshal protocols: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "alms://protocols",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})
}
