package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/ghassan/alms/internal/service"
)

// registerResources registers all MCP resource handlers on the server.
func (s *Server) registerResources(registry *service.Registry) {
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

	// alms://agents/{id} — single agent
	s.mcp.AddResource(mcp.NewResource(
		"alms://agents/{id}",
		"Agent Details",
		mcp.WithResourceDescription("Details of a specific agent"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		agentID, ok := req.Params.Arguments["id"]
		if !ok {
			return nil, fmt.Errorf("agent ID is required")
		}
		idStr, ok := agentID.(string)
		if !ok || idStr == "" {
			return nil, fmt.Errorf("agent ID is required")
		}

		agent, err := registry.Get(ctx, idStr)
		if err != nil {
			return nil, fmt.Errorf("get agent: %w", err)
		}

		data, err := json.Marshal(agent)
		if err != nil {
			return nil, fmt.Errorf("marshal agent: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      fmt.Sprintf("alms://agents/%s", idStr),
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
		health := map[string]any{
			"status":  "ok",
			"version": "0.1.0",
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
}
