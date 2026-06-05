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
	registerLearningStoreTools(s.mcp, learning)
	registerProtocolTools(s.mcp, syncer)
	registerProtocolStoreTools(s.mcp, learning)
	registerHealthTools(s.mcp, registry)
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

// registerLearningStoreTools registers Phase 2 learning tools.
func registerLearningStoreTools(mcpSrv *server.MCPServer, learning *service.Learning) {
	mcpSrv.AddTool(mcp.NewTool("learning.store",
		mcp.WithDescription("Create a new learning record with dedup check"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Source agent identifier")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Learning title")),
		mcp.WithString("body", mcp.Description("Learning body/content")),
		mcp.WithString("type", mcp.Required(), mcp.Description("Learning type: pattern, failure, config, protocol, edge_case")),
		mcp.WithArray("tags",
			mcp.Description("Tags for categorization"),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithString("supersedes", mcp.Description("Optional learning ID this supersedes")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agentID := req.GetString("agent_id", "")
		title := req.GetString("title", "")
		body := req.GetString("body", "")
		ltype := req.GetString("type", "")
		tags := req.GetStringSlice("tags", nil)
		supersedes := req.GetString("supersedes", "")

		if agentID == "" {
			return mcp.NewToolResultError("agent_id is required"), nil
		}

		rec := models.LearningRecord{
			Title:       title,
			Body:        body,
			Type:        models.LearningType(ltype),
			Tags:        tags,
			Author:      agentID,
			SrcAgentID:  agentID,
		}

		// Use dedup-aware store to check for exact and near duplicates
		id, dedupResult, err := learning.StoreLearningWithDedup(ctx, rec, supersedes)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(map[string]any{
			"learning_id":  id,
			"status":       "created",
			"is_duplicate": dedupResult.IsExactDup || dedupResult.IsNearDup,
		})
	})

	mcpSrv.AddTool(mcp.NewTool("learning.search",
		mcp.WithDescription("Full-text search for learnings via GIN index"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithString("type", mcp.Description("Optional learning type filter")),
		mcp.WithArray("tags",
			mcp.Description("Optional tag filters"),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithNumber("limit", mcp.Description("Max results (default 20)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := req.GetString("query", "")
		ltype := req.GetString("type", "")
		tags := req.GetStringSlice("tags", nil)
		limit := req.GetInt("limit", 20)

		if query == "" {
			return mcp.NewToolResultError("query is required"), nil
		}

		records, err := learning.Search(ctx, query, ltype, tags, limit)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if records == nil {
			records = []models.LearningRecord{}
		}

		return marshalResult(records)
	})

	mcpSrv.AddTool(mcp.NewTool("learning.delete",
		mcp.WithDescription("Soft-delete a learning record"),
		mcp.WithString("learning_id", mcp.Required(), mcp.Description("Learning ID to delete")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		learningID := req.GetString("learning_id", "")
		if learningID == "" {
			return mcp.NewToolResultError("learning_id is required"), nil
		}

		if err := learning.Delete(ctx, learningID); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(map[string]any{
			"deleted":     learningID,
			"status":      "soft_deleted",
		})
	})

	mcpSrv.AddTool(mcp.NewTool("learning.get",
		mcp.WithDescription("Get a single learning record by ID"),
		mcp.WithString("learning_id", mcp.Required(), mcp.Description("Learning ID to retrieve")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		learningID := req.GetString("learning_id", "")
		if learningID == "" {
			return mcp.NewToolResultError("learning_id is required"), nil
		}

		record, err := learning.Get(ctx, learningID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(record)
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

// registerProtocolStoreTools registers Phase 2 protocol tools.
func registerProtocolStoreTools(mcpSrv *server.MCPServer, learning *service.Learning) {
	mcpSrv.AddTool(mcp.NewTool("protocol.push",
		mcp.WithDescription("Create a new protocol"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Protocol title")),
		mcp.WithString("body", mcp.Description("Protocol body/content")),
		mcp.WithArray("target_tags",
			mcp.Description("Tags targeting which agents should receive this protocol"),
			mcp.Items(map[string]any{"type": "string"}),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		title := req.GetString("title", "")
		body := req.GetString("body", "")
		targetTags := req.GetStringSlice("target_tags", nil)

		if title == "" {
			return mcp.NewToolResultError("title is required"), nil
		}

		rec := models.ProtocolRecord{
			Title:       title,
			Body:        body,
			TargetTags:  targetTags,
			IsActive:    true,
			Version:     1,
		}

		id, err := learning.ProtocolPush(ctx, rec)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(map[string]any{
			"protocol_id": id,
			"status":      "created",
		})
	})

	mcpSrv.AddTool(mcp.NewTool("protocol.list",
		mcp.WithDescription("List all protocols"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		protocols, err := learning.ProtocolList(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if protocols == nil {
			protocols = []models.ProtocolRecord{}
		}

		return marshalResult(protocols)
	})
}

// registerHealthTools registers Phase 2 health tools.
func registerHealthTools(mcpSrv *server.MCPServer, registry *service.Registry) {
	mcpSrv.AddTool(mcp.NewTool("health.check",
		mcp.WithDescription("Check server health: PG ping + agent count"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Use registry to get actual agent count; short timeout for safety
		hcCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		count, countErr := registry.AgentCount(hcCtx)

		status := "ok"
		if countErr != nil {
			status = "degraded"
		}

		result := map[string]any{
			"status":      status,
			"agent_count": count,
			"version":     "0.1.0",
		}

		return marshalResult(result)
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
