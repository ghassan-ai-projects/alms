package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service"
)

func registerLearningSearchTool(mcpSrv *server.MCPServer, learning *service.Learning) {
	mcpSrv.AddTool(mcp.NewTool("learning.search",
		mcp.WithDescription("Full-text search for learnings via GIN index"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithString("type", mcp.Description("Optional learning type filter")),
		mcp.WithArray("tags",
			mcp.Description("Optional tag filters"),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithNumber("limit", mcp.Description("Max results (default 20)")),
		mcp.WithBoolean("include_enrichment", mcp.Description("Include enrichment metadata in results (default false)")),
		mcp.WithString("status", mcp.Description("Filter by enrichment status (default pending; use all to disable status filtering)")),
		mcp.WithBoolean("include_rejected", mcp.Description("Include rejected learnings in results (default false)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := req.GetString("query", "")
		ltype := req.GetString("type", "")
		tags := req.GetStringSlice("tags", nil)
		limit := req.GetInt("limit", 20)
		includeEnrichment := req.GetBool("include_enrichment", false)
		status := req.GetString("status", "pending")
		includeRejected := req.GetBool("include_rejected", false)

		if query == "" {
			return mcp.NewToolResultError("query is required"), nil
		}

		if status == "all" {
			status = ""
		}

		records, err := learning.SearchAdvanced(ctx, query, ltype, tags, limit, status, includeRejected)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if records == nil {
			records = []models.LearningRecord{}
		}

		// Strip enrichment from results unless requested
		if !includeEnrichment {
			filtered := make([]models.LearningRecord, len(records))
			for i, r := range records {
				r.EnrichmentMetadata = nil
				filtered[i] = r
			}
			records = filtered
		}

		return marshalResult(records)
	})
}
