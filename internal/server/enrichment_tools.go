package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/service"
)

// registerEnrichmentTools registers enrichment update tools.
func registerEnrichmentTools(mcpSrv *server.MCPServer, learning *service.Learning) {
	mcpSrv.AddTool(mcp.NewTool("learning.update_enrichment",
		mcp.WithDescription("Update enrichment metadata for a learning (used by OpenClaw async scoring)"),
		mcp.WithString("learning_id", mcp.Required(), mcp.Description("Learning ID to update")),
		mcp.WithObject("enrichment_patch",
			mcp.Required(),
			mcp.Description("JSON object to merge into enrichment_metadata"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		learningID := req.GetString("learning_id", "")
		enrichmentPatchRaw, ok := req.GetArguments()["enrichment_patch"]
		if !ok {
			return mcp.NewToolResultError("enrichment_patch is required"), nil
		}

		enrichmentJSON, err := json.Marshal(enrichmentPatchRaw)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal enrichment patch: %v", err)), nil
		}

		if err := learning.UpdateEnrichment(ctx, learningID, enrichmentJSON); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(map[string]any{
			"status":      "updated",
			"learning_id": learningID,
		})
	})
}
