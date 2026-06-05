package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service"
)

// registerTools registers all MCP tool handlers on the server.
func (s *Server) registerTools(registry *service.Registry, syncer *service.Syncer, learning *service.Learning) {
	registerAgentTools(s.mcp, registry)
	registerLearningTools(s.mcp, syncer)
	registerProtocolTools(s.mcp, syncer)
}

func registerAgentTools(mcpSrv *server.MCPServer, registry *service.Registry) {
	mcpSrv.AddTool(mcp.NewTool("agent.register",
		mcp.WithDescription("Register a new agent"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Unique agent identifier")),
		mcp.WithString("agent_type", mcp.Required(), mcp.Description("systemd or mcp_client")),
		mcp.WithString("display_name", mcp.Description("Human-readable name")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		agentID, _ := args["agent_id"].(string)
		agentTypeStr, _ := args["agent_type"].(string)
		displayName, _ := args["display_name"].(string)

		spec := models.AgentSpec{
			AgentID:     agentID,
			AgentType:   models.AgentType(agentTypeStr),
			DisplayName: displayName,
		}

		if err := registry.Register(ctx, spec); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(spec)
	})

	mcpSrv.AddTool(mcp.NewTool("agent.unregister",
		mcp.WithDescription("Unregister an existing agent"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Agent identifier to remove")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		agentID, _ := args["agent_id"].(string)

		spec, err := registry.Get(ctx, agentID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if err := registry.Delete(ctx, agentID); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(map[string]any{
			"deleted": agentID,
			"agent":   spec,
		})
	})

	mcpSrv.AddTool(mcp.NewTool("agent.update",
		mcp.WithDescription("Update an existing agent"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Agent identifier")),
		mcp.WithString("agent_type", mcp.Description("systemd or mcp_client")),
		mcp.WithString("display_name", mcp.Description("Human-readable name")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		agentID, _ := args["agent_id"].(string)
		agentTypeStr, _ := args["agent_type"].(string)
		displayName, _ := args["display_name"].(string)

		existing, err := registry.Get(ctx, agentID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("agent %s not found: %v", agentID, err)), nil
		}

		if displayName != "" {
			existing.DisplayName = displayName
		}
		if agentTypeStr != "" {
			existing.AgentType = models.AgentType(agentTypeStr)
		}

		if err := registry.Update(ctx, agentID, existing); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(existing)
	})

	mcpSrv.AddTool(mcp.NewTool("agent.list",
		mcp.WithDescription("List registered agents"),
		mcp.WithString("agent_type", mcp.Description("Optional type filter")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 100)")),
		mcp.WithNumber("offset", mcp.Description("Result offset (default 0)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agentType := req.GetString("agent_type", "")
		limit := req.GetInt("limit", 100)
		offset := req.GetInt("offset", 0)

		filter := make(map[string]string)
		if agentType != "" {
			filter["agent_type"] = agentType
		}

		agents, err := registry.List(ctx, filter, limit, offset)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if agents == nil {
			agents = []models.AgentSpec{}
		}

		return marshalResult(agents)
	})

	mcpSrv.AddTool(mcp.NewTool("agent.heartbeat",
		mcp.WithDescription("Send a heartbeat for an agent"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Agent identifier")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		agentID, _ := args["agent_id"].(string)

		ts, err := registry.Heartbeat(ctx, agentID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(map[string]any{
			"agent_id":         agentID,
			"last_heartbeat": ts.Format(time.RFC3339),
		})
	})
}

func registerLearningTools(mcpSrv *server.MCPServer, syncer *service.Syncer) {
	mcpSrv.AddTool(mcp.NewTool("learning.sync",
		mcp.WithDescription("Sync new learnings for an agent"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Agent identifier")),
		mcp.WithString("since", mcp.Description("RFC3339 timestamp to sync from")),
		mcp.WithString("type", mcp.Description("Optional learning type filter")),
		mcp.WithArray("tags",
			mcp.Description("Optional tag filters"),
			mcp.Items(map[string]any{"type": "string"}),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agentID := req.GetString("agent_id", "")
		sinceStr := req.GetString("since", "")
		ltype := req.GetString("type", "")
		tags := req.GetStringSlice("tags", nil)

		var since time.Time
		if sinceStr != "" {
			var err error
			since, err = time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid since timestamp: %v", err)), nil
			}
		} else {
			since = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		}

		records, err := syncer.Sync(ctx, agentID, since, ltype, tags)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if records == nil {
			records = []models.LearningRecord{}
		}

		return marshalResult(records)
	})

	mcpSrv.AddTool(mcp.NewTool("learning.sync_ack",
		mcp.WithDescription("Acknowledge received learnings (gap-safe)"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Agent identifier")),
		mcp.WithArray("learning_ids",
			mcp.Required(),
			mcp.Description("Ordered list of acknowledged learning IDs"),
			mcp.Items(map[string]any{"type": "string"}),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agentID := req.GetString("agent_id", "")
		ids := req.GetStringSlice("learning_ids", nil)

		if len(ids) == 0 {
			return mcp.NewToolResultError("learning_ids is required"), nil
		}

		if err := syncer.SyncAck(ctx, agentID, ids); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(map[string]any{
			"status": "acknowledged",
			"count":  len(ids),
		})
	})
}

func registerProtocolTools(mcpSrv *server.MCPServer, syncer *service.Syncer) {
	mcpSrv.AddTool(mcp.NewTool("protocol.pull",
		mcp.WithDescription("Pull active protocols for an agent"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Agent identifier")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		agentID, _ := args["agent_id"].(string)

		protocols, err := syncer.PullProtocols(ctx, agentID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if protocols == nil {
			protocols = []models.ProtocolRecord{}
		}

		return marshalResult(protocols)
	})

	mcpSrv.AddTool(mcp.NewTool("protocol.pull_since",
		mcp.WithDescription("Pull protocols created after a given protocol ID"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Agent identifier")),
		mcp.WithString("since_id", mcp.Required(), mcp.Description("Protocol ID to pull since")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		agentID, _ := args["agent_id"].(string)
		sinceID, _ := args["since_id"].(string)

		protocols, err := syncer.PullProtocolsSince(ctx, agentID, sinceID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if protocols == nil {
			protocols = []models.ProtocolRecord{}
		}

		return marshalResult(protocols)
	})
}

// marshalResult converts a value to JSON and wraps it in a CallToolResult.
func marshalResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
